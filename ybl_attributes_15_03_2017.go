package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	//"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/crypto/attr"
)

//==============================================================================================================================
//	 Participant types - Each participant type is mapped to an integer which we use to compare to the value stored in a
//						 user's eCert
//==============================================================================================================================
const ROLE_ADMIN = "assigner"
const ROLE_ANCHOR = "Anchor"
const ROLE_VENDOR = "Vendor"
const ROLE_PAYMENT_MAKER = "PaymentMaker"
const ROLE_PAYMENT_CHECKER = "PaymentChecker"

//==============================================================================================================================
//	 Status types - Anchor Program lifecycle is broken down into 8 statuses, this is part of the business logic to determine what can
//					be done to the Credential at points in it's lifecycle
//==============================================================================================================================
const STATE_TEMPLATE = 0
const STATE_PROGRAM_INITIATED = 1
const STATE_PURCHASE_ORDER_PLACED = 2
const STATE_INVOICE_RAISED = 3
const STATE_VENDOR_INVOICE_APPROVED = 4
const STATE_ANCHOR_AUTHORISED_INVOICE_PAYMENT = 5
const STATE_INVOICE_PAYMENT_REQUESTED = 6
const STATE_INVOICE_PAYMENT_INITIATED = 7
const STATE_INVOICE_PAYMENT_PENDING_APPROVAL = 8
const STATE_INVOICE_PAYMENT_APPROVED = 9
const STATE_INVOICE_PAID = 10
const STATE_INVOICE_SETTLED = 11
const STATE_ANCHOR_PROGRAM_CLOSED = 12
const STATE_INVOICE_RETIRED = 20

//==============================================================================================================================
//	 Structure Definitions
//==============================================================================================================================
//	Chaincode - A blank struct for use with Shim (A HyperLedger included go file used for get/put state
//				and other HyperLedger functions)
//==============================================================================================================================
type AssetManagementChaincode struct {
}

//==============================================================================================================================
//	AnchorProgram - Defines the structure for an AnchorProgram object. JSON on right tells it what JSON fields to map to
//			  that element when reading a JSON object into the struct e.g. JSON name -> Struct Name.
//==============================================================================================================================
type AnchorProgram struct {
	Owner                     string      `json:"owner"`
	POraisedBy                string      `json:"poraisedby"`
	PORaisedAgainst           string      `json:"poraisedAgainst"`
	AnchorName                string      `json:"anchorname"`
	AnchorID                  string      `json:"anchorid"`
	AnchorAccountNo           string      `json:"anchoraccountno"`
	AnchorPOAmount            float64     `json:"anchorpoamount"`
	AnchorIFSCCode            string      `json:"anchorifsc"`
	AnchorAgreement           string      `json:"anchorAgreement"`
	VendorAgreement           string      `json:"vendorAgreement"`
	AnchorLimit               float64     `json:"anchorlimit"`
	AnchorExpiryDate          string      `json:"anchorexpdate"`
	AnchorInterest            string      `json:"anchorinterest"`
	AnchorGarceInterest       string      `json:"anchorgraceinterest"`
	AnchorGarceInterestperiod string      `json:"anchorgraceinterestperiod"`
	AnchorPenalInterest       string      `json:"anchorpenalinterest"`
	AnchorLiquidation         string      `json:"anchorliquidation"`
	AnchorPoImage             string      `json:"anchorpoimage"`
	AnchorPoID                string      `json:"anchorpoid"`
	VendorID                  string      `json:"vendorid"`
	VendorFName               string      `json:"vendorfname"`
	VendorLName               string      `json:"vendorlname"`
	Vendoremail               string      `json:"vendoremail"`
	Vendorphone               string      `json:"vendorphone"`
	Vendorpanno               string      `json:"vendorpanno"`
	Vendoraddress             string      `json:"vendoraddress"`
	Vendorbank                string      `json:"vendorbank"`
	Vendorbaddress            string      `json:"vendorbaddress"`
	Vendoraccountno           string      `json:"vendoraccountno"`
	Vendorifsccode            string      `json:"vendorifsccode"`
	Vendorlimit               float64     `json:"vendorlimit"`
	VendorExpirydate          string      `json:"vendorexpirydate"`
	//POTimestamp               time.Time   `json:"poRaisedTime"`
	POAcknowledged            bool        `json:"poacknowledged"`
	Items                     []MyBoxItem `json:"invoices"`
	Status                    int         `json:"status"`
	AnchorProgramID           string      `json:"anchorprogramID"`
	PoForks                   []string    `json:"poForks"`
	POParent                  string      `json:"poParent"`
	PORemarks                 string      `json:"poRemarks"`
	Settled                   bool        `json:"settled"`
}

//==============================================================================================================================
//	MyBoxItem - Defines the structure for an invoice object. JSON on right tells it what JSON fields to map to
//			  that element when reading a JSON object into the struct e.g. JSON name -> Struct Name.
//==============================================================================================================================
type MyBoxItem struct {
	POID                   string    `json:"poIDr"`
	MOID                   string    `json:"moID"`
	MOOwner                string    `json:"moOwner"`
	AnchorName             string    `json:"anchorname"`
	AnchorAccountNo        string    `json:"anchoraccountno"`
	AnchorPOAmount         float64   `json:"anchorpoamount"`
	AnchorIFSCCode         string    `json:"anchorifsc"`
	AnchorInterest         string    `json:"anchorinterest"`
	InvoiceRaisedBy        string    `json:"invraisedby"`
	InvoiceRaisedAgainst   string    `json:"invraisedAgainst"`
	MOAmount               float64   `json:"moAmount"`
	InvoiceID              string    `json:"invoiceid"`
	InvoiceImage           string    `json:"invoiceimage"`
	Vendorfname            string    `json:"vendorFname"`
	Vendorbank             string    `json:"vendorBank"`
	Vendorifsccode         string    `json:"venDorbank"`
	AnchorPoID             string    `json:"anchOrPOID"`
	ApprovedInvoiceAmount  float64   `json:"approvedinvoiceAmount"`
	MOStatus               int       `json:"moStatus"`
	SettlementAmount       string    `json:"settlementAmount"`
	CheckerApprovedPayment bool      `json:"checkerApprovedPayment"`
	//MOTimestamp            time.Time `json:"invoiceRaisedTime"`
	MOReceivableAmount     float64   `json:"moReceivableAmount"`
	PaymentChannel         string    `json:"paymentChannel"`
	MOPaid                 bool      `json:"mopaid"`
	TxnStatus              string    `json:"txnStatus"`
	UTRNumber              string    `json:"utrnumber"`
	MOForks                []string  `json:"moForks"`
	MOParent               string    `json:"moParent"`
	MoOriginal             float64   `json:"moOriginalAmount"`
	MORemarks              string    `json:"moRemarks"`
	MOSettled              bool      `json:"mosettled"`
}

//==============================================================================================================================
//	Anchor Program Holder - Defines the structure that holds all the anchorIDs for Anchor Program records that have been created.
//				Used as an index when querying all anchor records.
//==============================================================================================================================

type Anchor_Program_Holder struct {
	ANCHOR_PROGRAMs []string `json:"anchorprograms"`
}

//==============================================================================================================================
//	ProgramIDs - Defines the structure that holds all the IDs for Anchor Program records that have been created.
//				Used as an index when querying all anchor records.
//==============================================================================================================================

type ProgramIDs struct {
	POraisedBy                string    `json:"poraisedby"`
	AnchorName                string    `json:"anchorname"`
	AnchorID                  string    `json:"anchorid"`
	AnchorAccountNo           string    `json:"anchoraccountno"`
	AnchorPOAmount            float64   `json:"anchorpoamount"`
	AnchorIFSCCode            string    `json:"anchorifsc"`
	AnchorLimit               float64   `json:"anchorlimit"`
	AnchorExpiryDate          string    `json:"anchorexpdate"`
	AnchorInterest            string    `json:"anchorinterest"`
	AnchorGarceInterest       string    `json:"anchorgraceinterest"`
	AnchorGarceInterestperiod string    `json:"anchorgraceinterestperiod"`
	AnchorPenalInterest       string    `json:"anchorpenalinterest"`
	AnchorLiquidation         string    `json:"anchorliquidation"`
	AnchorPoID                string    `json:"anchorpoid"`
	VendorID                  string    `json:"vendorid"`
	VendorFName               string    `json:"vendorfname"`
	VendorLName               string    `json:"vendorlname"`
	Vendoremail               string    `json:"vendoremail"`
	Vendorphone               string    `json:"vendorphone"`
	Vendorpanno               string    `json:"vendorpanno"`
	Vendoraddress             string    `json:"vendoraddress"`
	Vendorbank                string    `json:"vendorbank"`
	Vendorbaddress            string    `json:"vendorbaddress"`
	Vendoraccountno           string    `json:"vendoraccountno"`
	Vendorifsccode            string    `json:"vendorifsccode"`
	Vendorlimit               float64   `json:"vendorlimit"`
	VendorExpirydate          string    `json:"vendorexpirydate"`
	//POTimestamp               time.Time `json:"poRaisedTime"`
	POAcknowledged            bool      `json:"poacknowledged"`
	Invoices                  []InvoiceIDs
	Status                    int    `json:"status"`
	AnchorProgramID           string `json:"anchorprogramID"`
	Settled                   bool   `json:"settled"`
}

//==============================================================================================================================
//	Invoice Holder - Defines the structure that holds all the invoiceIDs for Invoice records that have been created.
//				Used as an index when querying all invoice records.
//==============================================================================================================================

type Invoice_Holder struct {
	INVOICEs []string `json:"invoices"`
}

//==============================================================================================================================
//	InvoiceIDs - Defines the structure that holds all the IDs for invoice records that have been created.
//				Used as an index when querying all invoice records.
//==============================================================================================================================

type InvoiceIDs struct {
	POID                   string    `json:"poIDr"`
	MOID                   string    `json:"moID"`
	MOOwner                string    `json:"moOwner"`
	AnchorName             string    `json:"anchorname"`
	AnchorAccountNo        string    `json:"anchoraccountno"`
	AnchorPOAmount         float64   `json:"anchorpoamount"`
	AnchorIFSCCode         string    `json:"anchorifsc"`
	AnchorInterest         string    `json:"anchorinterest"`
	InvoiceRaisedBy        string    `json:"invraisedby"`
	InvoiceRaisedAgainst   string    `json:"invraisedAgainst"`
	MOAmount               float64   `json:"moAmount"`
	InvoiceID              string    `json:"invoiceid"`
	Vendorfname            string    `json:"vendorFname"`
	Vendorbank             string    `json:"vendorBank"`
	Vendorifsccode         string    `json:"venDorbank"`
	AnchorPoID             string    `json:"anchOrPOID"`
	ApprovedInvoiceAmount  float64   `json:"approvedinvoiceAmount"`
	MOStatus               int       `json:"moStatus"`
	CheckerApprovedPayment bool      `json:"checkerApprovedPayment"`
	//MOTimestamp            time.Time `json:"invoiceRaisedTime"`
	MOReceivableAmount     float64   `json:"moReceivableAmount"`
	PaymentChannel         string    `json:"paymentChannel"`
	MOPaid                 bool      `json:"mopaid"`
	UTRNumber              string    `json:"utrnumber"`
	MOSettled              bool      `json:"mosettled"`
}

//==============================================================================================================================
//	Init Function - Called when the user deploys the chaincode
//==============================================================================================================================
func (t *AssetManagementChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	//	myLogger.Info("[AssetManagementChaincode] Init")

	var anchorProgramIDs Anchor_Program_Holder
	var invoiceIDs Invoice_Holder

	bytes, err := json.Marshal(anchorProgramIDs)
	if err != nil {
		return nil, errors.New("Error creating Anchor_Program_Holder record")
	}

	err = stub.PutState("anchorProgramIDs", bytes)

	bites, err := json.Marshal(invoiceIDs)
	if err != nil {
		return nil, errors.New("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bites)

	if len(args) != 0 {
		return nil, errors.New("Incorrect number of arguments. Expecting 0")
	}

	// Set the role of the users that are allowed to assign assets
	// The metadata will contain the role of the users that are allowed to assign assets
	assignerRole, err := stub.GetCallerMetadata()
	fmt.Printf("Assiger role is %v\n", string(assignerRole))

	if err != nil {
		return nil, fmt.Errorf("Failed getting metadata, [%v]", err)
	}

	if len(assignerRole) == 0 {
		return nil, errors.New("Invalid assigner role. Empty.")
	}

	stub.PutState("assignerRole", assignerRole)

	return nil, nil
}

//==============================================================================================================================
//	 retrieve_invoice - Gets the state of the data at MOID in the ledger then converts it from the stored
//					JSON into the MyBoxItem struct for use in the contract. Returns the MyBoxItem struct.
//					Returns empty v if it errors.
//==============================================================================================================================
func (t *AssetManagementChaincode) retrieve_invoice(stub shim.ChaincodeStubInterface, moid string) (MyBoxItem, error) {

	var x MyBoxItem

	bytes, err := stub.GetState(moid)
	if err != nil {
		fmt.Printf("RETRIEVE_INVOICE: Failed to invoke order_code: %s", err)
		return x, errors.New("RETRIEVE_INVOICE: Error retrieving order with InvoiceID = " + moid)
	}

	err = json.Unmarshal(bytes, &x)
	if err != nil {
		fmt.Printf("RETRIEVE_INVOICE: Corrupt invoice record "+string(bytes)+": %s", err)
		return x, errors.New("RETRIEVE_INVOICE: Corrupt invoice record" + string(bytes))
	}

	return x, nil
}

//==============================================================================================================================
//	 retrieve_anchorprogram - Gets the state of the data at poID in the ledger then converts it from the stored
//					JSON into the AnchorProgram struct for use in the contract. Returns the AnchorProgram struct.
//					Returns empty v if it errors.
//==============================================================================================================================
func (t *AssetManagementChaincode) retrieve_anchorprogram(stub shim.ChaincodeStubInterface, anchorProgramID string) (AnchorProgram, error) {

	var v AnchorProgram

	bytes, err := stub.GetState(anchorProgramID)
	if err != nil {
		fmt.Printf("RETRIEVE_ANCHORPROGRAM: Failed to invoke order_code: %s", err)
		return v, errors.New("RETRIEVE_ANCHORPROGRAM: Error retrieving order with AnchorProgramID = " + anchorProgramID)
	}

	err = json.Unmarshal(bytes, &v)
	if err != nil {
		fmt.Printf("RETRIEVE_ANCHORPROGRAM: Corrupt anchorprogram record "+string(bytes)+": %s", err)
		return v, errors.New("RETRIEVE_ANCHORPROGRAM: Corrupt anchorprogram record" + string(bytes))
	}

	return v, nil
}

//==============================================================================================================================
// copy_invoice - Copies the MyBoxItem struct as revision. MyBox struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (t *AssetManagementChaincode) copy_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem) (MyBoxItem, error) {

	var box MyBoxItem
	box = x
	box.MOID = x.MOID + "-R1"
	//userb.count = userb.count + 1
	return box, nil

}

//==============================================================================================================================
// copy_anchorprogram - Copies the Anchorprogram struct as revision. struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (t *AssetManagementChaincode) copy_anchorprogram(stub shim.ChaincodeStubInterface, v AnchorProgram) (AnchorProgram, error) {

	var pobox AnchorProgram
	pobox = v
	pobox.AnchorProgramID = v.AnchorProgramID + "-R1"
	//userb.count = userb.count + 1
	return pobox, nil

}

//==============================================================================================================================
// save_invoice - Writes to the ledger the MyBox struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (t *AssetManagementChaincode) save_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem) (bool, error) {

	bytes, err := json.Marshal(x)
	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error converting Invoice record: %s", err)
		return false, errors.New("Error converting Invoice record")
	}

	err = stub.PutState(x.MOID, bytes)
	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error storing Invoice record: %s", err)
		return false, errors.New("Error storing Invoice record")
	}

	return true, nil
}

//==============================================================================================================================
// save_changes - Writes to the ledger the Anchor Program struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (t *AssetManagementChaincode) save_changes(stub shim.ChaincodeStubInterface, v AnchorProgram) (bool, error) {

	bytes, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error converting Anchor Progam record: %s", err)
		return false, errors.New("Error converting Anchor Progam record")
	}

	err = stub.PutState(v.AnchorProgramID, bytes)
	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error storing Anchor Progam record: %s", err)
		return false, errors.New("Error storing Anchor Progam record")
	}

	return true, nil
}

//==============================================================================================================================
//	 Router Functions
//==============================================================================================================================
//	Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		  initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *AssetManagementChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	callerRole, err := stub.ReadCertAttribute("role")
	if err != nil {
		fmt.Printf("Error reading attribute 'role' [%v] \n", err)
		return nil, fmt.Errorf("Failed fetching caller role. Error was [%v]", err)
	}
	caller_affiliation := string(callerRole[:])

	callerAccount, err := stub.ReadCertAttribute("account")
	if err != nil {
		return nil, fmt.Errorf("Failed fetching caller account. Error was [%v]", err)
	}
	if function == "create_anchorprogram" {
		return t.create_anchorprogram(stub, callerAccount, caller_affiliation, args[0])
	} else { // If the function is not a create then there must be a order so we need to retrieve the order.

		argPos := 0

		v, err := t.retrieve_anchorprogram(stub, args[argPos])
		if err != nil {
			fmt.Printf("INVOKE: Error retrieving Anchor Program: %s", err)
			return nil, errors.New("Error retrieving Anchor Program")
		}

		if strings.Contains(function, "update") == false && function != "settlement_anchorprogram" { // If the function is not an update or a scrappage it must be a transfer so we need to get the ecert of the recipient.
			receiverCert, err := base64.StdEncoding.DecodeString(args[1])
			if err != nil {
				fmt.Printf("Error decoding [%v] \n", err)
				return nil, errors.New("Failed decodinf owner")
			}

			receiverAcc, err := attr.GetValueFrom("account", receiverCert)
			if err != nil {
				fmt.Printf("Error reading account [%v] \n", err)
				return nil, fmt.Errorf("Failed fetching recipient account. Error was [%v]", err)
			}

			receiverAccount := string(receiverAcc[:])

			recRole, err := attr.GetValueFrom("role", receiverCert)
			if err != nil {
				fmt.Printf("Error reading account [%v] \n", err)
				return nil, fmt.Errorf("Failed fetching recipient role. Error was [%v]", err)
			}

			rec_affiliation := string(recRole[:])

			if function == "admin_to_anchor" {
				return t.admin_to_anchor(stub, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "anchor_to_admin_rev" {
				return t.anchor_to_admin_rev(stub, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation, args[2])
			} else if function == "anchor_to_vendor" {
				return t.anchor_to_vendor(stub, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "vendor_to_anchor_rev" {
				return t.vendor_to_anchor_rev(stub, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation, args[2])
			} else if function == "transfer_vendor_to_anchor_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_vendor_to_anchor_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "transfer_rev_anchor_to_vendor_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_rev_anchor_to_vendor_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation, args[3])
			} else if function == "transfer_anchor_to_vendor_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_anchor_to_vendor_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "transfer_rev_vendor_to_anchor_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_rev_vendor_to_anchor_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation, args[3])
			} else if function == "transfer_vendor_to_admin_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_vendor_to_admin_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "transfer_rev_admin_to_vendor_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_rev_admin_to_vendor_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation, args[3])
			} else if function == "transfer_admin_to_payment_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_admin_to_payment_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "transfer_rev_payment_to_admin_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_rev_payment_to_admin_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation, args[3])
			} else if function == "transfer_payment_maker_to_payment_checker_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_payment_maker_to_payment_checker_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "transfer_rev_payment_checker_to_payment_maker_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_rev_payment_checker_to_payment_maker_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation, args[3])
			} else if function == "transfer_payment_checker_to_payment_maker_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_payment_checker_to_payment_maker_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} /*else if function == "transfer_payment_checker_to_anchor_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_payment_checker_to_anchor_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			} else if function == "transfer_anchor_to_payment_checker_invoice" {
				x, err := t.retrieve_invoice(stub, args[2])
				if err != nil {
					fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
					return nil, errors.New("Error retrieving INVOICE")
				}
				return t.transfer_anchor_to_payment_checker_invoice(stub, x, v, []byte(callerAccount), string(caller_affiliation), receiverAccount, rec_affiliation)
			}
			*/

			//transfer payment checker to anchor invoice
			//transfer anchor to payment checker invoice

		} else if function == "update_anchor_details" {
			return t.update_anchor_details(stub, v, callerAccount, caller_affiliation, args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12])
		} else if function == "update_vendor_details" {
			return t.update_vendor_details(stub, v, callerAccount, caller_affiliation, args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14])
		} else if function == "update_anchor_purchase_order" {
			return t.update_anchor_purchase_order(stub, v, callerAccount, caller_affiliation, args[1], args[2], args[3])
		} else if function == "settlement_anchorprogram" {
			return t.settlement_anchorprogram(stub, v, callerAccount, caller_affiliation)
		} else if function == "update_vendor_po_acknowledgement" {
			return t.update_vendor_po_acknowledgement(stub, v, callerAccount, caller_affiliation)
		} else if function == "update_vendor_create_invoice" {
			return t.update_vendor_create_invoice(stub, v, callerAccount, caller_affiliation, args[1])
		} else if function == "update_vendor_invoice_details" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_vendor_invoice_details(stub, x, v, callerAccount, caller_affiliation, args[2], args[3], args[4])
		} else if function == "update_anchor_invoice_authorized_amount" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_anchor_invoice_authorized_amount(stub, x, v, callerAccount, caller_affiliation, args[2])
		} else if function == "update_maker_invoice_payment" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_maker_invoice_payment(stub, x, v, callerAccount, caller_affiliation, args[2], args[3])
		} else if function == "update_checker_invoice_approval" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_checker_invoice_approval(stub, x, v, callerAccount, caller_affiliation)
		} else if function == "update_rev_checker_invoice_approval" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_rev_checker_invoice_approval(stub, x, v, callerAccount, caller_affiliation, args[2])
		} else if function == "update_checker_invoice_payment" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_checker_invoice_payment(stub, x, v, callerAccount, caller_affiliation, args[2], args[3])
		} else if function == "update_rev_checker_invoice_payment" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_rev_checker_invoice_payment(stub, x, v, callerAccount, caller_affiliation, args[2])
		} else if function == "update_checker_invoice_settlement" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_checker_invoice_settlement(stub, x, v, callerAccount, caller_affiliation, args[2])
		} else if function == "update_rev_checker_invoice_settlement" {
			x, err := t.retrieve_invoice(stub, args[1])
			if err != nil {
				fmt.Printf("INVOKE: A Error retrieving Invoice: %s", err)
				return nil, errors.New("Error retrieving INVOICE")
			}
			return t.update_rev_checker_invoice_settlement(stub, x, v, callerAccount, caller_affiliation, args[2])
		}

		return nil, errors.New("Function of that name doesn't exist.")
	}

}

//=================================================================================================================================
//	 Create Function
//=================================================================================================================================
//	 Create AnchorProgram - Creates the initial JSON for the order and then saves it to the ledger.
//=================================================================================================================================
func (t *AssetManagementChaincode) create_anchorprogram(stub shim.ChaincodeStubInterface, callerAccount []byte, caller_affiliation string, anchorprogramID string) ([]byte, error) {
	var v AnchorProgram

	v.AnchorProgramID = anchorprogramID
	v.Owner = string(callerAccount)

	/*matched, err := regexp.Match("^[A-z][A-z][0-9]{7}", []byte(anchorprogramID)) // matched = true if the poID passed fits format of two letters followed by seven digits
	if err != nil {
		fmt.Printf("CREATE_ANCHORPROGRAM: Invalid anchorprogramID: %s", err)
		return nil, errors.New("Invalid anchorprogramID")
	}*/

	if v.AnchorProgramID == "" { /*||
		matched == false */
		fmt.Printf("CREATE_ANCHORPROGRAM: Invalid anchorprogramID provided")
		return nil, errors.New("Invalid poID provided")
	}

	record, err := stub.GetState(v.AnchorProgramID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
	if record != nil {
		return nil, errors.New("AnchorProgram already exists")
	}

	// Recover the role that is allowed to create  assets
	assignerRole, err := stub.GetState("assignerRole")
	if err != nil {
		fmt.Printf("Error getting role [%v] \n", err)
		return nil, errors.New("Failed fetching assigner role")
	}

	assigner := string(assignerRole[:])

	if caller_affiliation != assigner {
		fmt.Printf("Caller is not assigner - caller %v assigner %v\n", caller_affiliation, assigner)
		return nil, fmt.Errorf("The caller does not have the rights to invoke assign. Expected role [%v], caller role [%v]", assigner, caller_affiliation)
	}

	_, err = t.save_changes(stub, v)
	if err != nil {
		fmt.Printf("CREATE_ANCHORPROGRAM: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	bytes, err := stub.GetState("anchorProgramIDs")
	if err != nil {
		return nil, errors.New("Unable to get anchorProgramIDs")
	}

	var anchorProgramIDs Anchor_Program_Holder

	err = json.Unmarshal(bytes, &anchorProgramIDs)
	if err != nil {
		return nil, errors.New("Corrupt Anchor_Program_Holder record")
	}

	anchorProgramIDs.ANCHOR_PROGRAMs = append(anchorProgramIDs.ANCHOR_PROGRAMs, anchorprogramID)

	bytes, err = json.Marshal(anchorProgramIDs)
	if err != nil {
		fmt.Print("Error creating Anchor_Program_Holder record")
	}

	err = stub.PutState("anchorProgramIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	//	myLogger.Debugf("New owner of [%s] is [% x]", v, v.Owner)

	return nil, nil

}

//=================================================================================================================================
//	 Create Invoice - Creates the initial JSON for the invoice and then saves it to the ledger.
//=================================================================================================================================
func (t *AssetManagementChaincode) update_vendor_create_invoice(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string, id string) ([]byte, error) {

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		v.Owner == string(callerAccount) &&
		caller_affiliation == ROLE_VENDOR &&
		//v.POAcknowledged == true &&
		v.Settled == false { // If the roles and users are ok

		var item MyBoxItem

		//	id := randStr(9, "alphanum")
		item.POID = v.AnchorProgramID
		item.MOID = id
		item.AnchorName = v.AnchorName
		item.AnchorAccountNo = v.AnchorAccountNo
		item.AnchorPOAmount = v.AnchorPOAmount
		item.AnchorIFSCCode = v.AnchorIFSCCode
		item.AnchorInterest = v.AnchorInterest
		item.Vendorfname = v.VendorFName
		item.Vendorbank = v.Vendorbank
		item.Vendorifsccode = v.Vendorifsccode
		item.AnchorPoID = v.AnchorPoID

		item.MOOwner = string(callerAccount)
		item.InvoiceRaisedBy = string(callerAccount)

		/*matched, err := regexp.Match("^[A-z][A-z][0-9]{7}", []byte(id)) // matched = true if the poID passed fits format of two letters followed by seven digits
		if err != nil {
			fmt.Printf("CREATE_INVOICE: Invalid InvoiceID: %s", err)
			return nil, errors.New("Invalid InvoiceID")
		}*/

		if item.MOID == "" {
			fmt.Printf("CREATE_INVOICE: Invalid InvoiceID provided")
			return nil, errors.New("Invalid moID provided")
		}

		record, err := stub.GetState(item.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

		if caller_affiliation != ROLE_VENDOR { // Only the vendor can create a new Invoice template
			return nil, errors.New("Permission Denied")
		}

		_, err = t.save_invoice(stub, item)
		if err != nil {
			fmt.Printf("CREATE_INVOICE: Error saving changes: %s", err)
			return nil, errors.New("Error saving changes")
		}

		bytes, err := stub.GetState("invoiceIDs")
		if err != nil {
			return nil, errors.New("Unable to get invoiceIDs")
		}

		var invoiceIDs Invoice_Holder

		err = json.Unmarshal(bytes, &invoiceIDs)
		if err != nil {
			return nil, errors.New("Corrupt Invoice_Holder record")
		}

		invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, id)

		bytes, err = json.Marshal(invoiceIDs)
		if err != nil {
			fmt.Print("Error creating Invoice_Holder record")
		}

		err = stub.PutState("invoiceIDs", bytes)
		if err != nil {
			return nil, errors.New("Unable to put the state")
		}

		v.Items = append(v.Items, (item))

	} else { // Otherwise if there is an error

		fmt.Printf("CREATE_INVOICE: Permission Denied")
		return nil, errors.New("Permission Denied")

	}
	_, err := t.save_changes(stub, v)
	if err != nil {
		fmt.Printf("CREATE_INVOICE: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes to Anchor Program")
	}

	return nil, nil

}

//=================================================================================================================================
//	 Transfer Functions
//=================================================================================================================================
//	 admin_to_anchor
//=================================================================================================================================
func (t *AssetManagementChaincode) admin_to_anchor(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {
	if v.VendorFName == "UNDEFINED" ||
		v.VendorLName == "UNDEFINED" ||
		v.Vendorphone == "UNDEFINED" ||
		v.Vendoraddress == "UNDEFINED" ||
		v.Vendoremail == "UNDEFINED" ||
		v.Vendorlimit == 0 ||
		v.AnchorName == "UNDEFINED" ||
		v.AnchorID == "UNDEFINED" ||
		v.AnchorAccountNo == "UNDEFINED" ||
		v.AnchorIFSCCode == "UNDEFINED" ||
		v.AnchorLimit == 0 ||
		v.AnchorExpiryDate == "UNDEFINED" ||
		v.AnchorInterest == "UNDEFINED" ||
		v.AnchorGarceInterest == "UNDEFINED" ||
		v.AnchorGarceInterestperiod == "UNDEFINED" ||
		v.AnchorPenalInterest == "UNDEFINED" ||
		v.AnchorLiquidation == "UNDEFINED" { //If any part of the order is undefined it has not been fully manufacturered so cannot be sent

		fmt.Printf("ADMIN_TO_ANCHOR: AnchorProgram not fully defined")

		return nil, errors.New("AnchorProgram not fully defined")
	}

	// Verify the identity of the caller
	// Only the owner can transfer one of his assets

	prvOwner := []byte(v.Owner)

	// Verify ownership

	if bytes.Compare(prvOwner, callerAccount) != 0 {
		return nil, fmt.Errorf("Failed verifying caller ownership.")
	}

	if v.Status == STATE_TEMPLATE &&
		caller_affiliation == ROLE_ADMIN &&
		recipient_affiliation == ROLE_ANCHOR &&
		v.Settled == false { // If the roles and users are ok

		v.Owner = receiverAccount          // then make the owner the new owner
		v.Status = STATE_PROGRAM_INITIATED // and mark it in the state of creating purchase order
		v.POraisedBy = receiverAccount

	} else { // Otherwise if there is an error

		fmt.Printf("ADMIN_TO_ANCHOR: Permission Denied")
		return nil, errors.New("Permission Denied")

	}

	_, err := t.save_changes(stub, v) // Write new state

	if err != nil {
		fmt.Printf("ADMIN_TO_ANCHOR: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	return nil, nil // We are Done

}

//=================================================================================================================================
//	 anchor_to_admin_rev
//=================================================================================================================================
func (t *AssetManagementChaincode) anchor_to_admin_rev(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string, new_value string) ([]byte, error) {

	// Verify the identity of the caller
	// Only the owner can transfer one of his assets

	prvOwner := []byte(v.Owner)

	// Verify ownership

	if bytes.Compare(prvOwner, callerAccount) != 0 {
		return nil, fmt.Errorf("Failed verifying caller ownership.")
	}

	var pobox AnchorProgram
	pobox = v

	if v.Status == STATE_PROGRAM_INITIATED &&
		caller_affiliation == ROLE_ANCHOR &&
		recipient_affiliation == ROLE_ADMIN &&
		v.Settled == false { // If the roles and users are ok

		pobox.Owner = receiverAccount // then make the owner the new owner
		pobox.Status = STATE_TEMPLATE // and mark it in the state of creating purchase order
		pobox.AnchorProgramID = v.AnchorProgramID + "-R1"
		pobox.AnchorPOAmount = 0 // Update to the new value
		pobox.AnchorPoImage = "UNDEFINED"
		pobox.AnchorPoID = "UNDEFINED"
		pobox.POParent = v.AnchorProgramID
		pobox.PORemarks = new_value
		v.PoForks = append(v.PoForks, pobox.AnchorProgramID)

		record, _ := stub.GetState(pobox.AnchorProgramID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("AnchorProgramID already exists")
		}

	} else { // Otherwise if there is an error

		fmt.Printf("ANCHOR_TO_ADMIN_REV: Permission Denied")
		return nil, errors.New("Permission Denied")

	}

	_, err := t.save_changes(stub, pobox) // Write new state

	if err != nil {
		fmt.Printf("ANCHOR_TO_ADMIN_REV: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	_, errs := t.save_changes(stub, v) // Write new state

	if errs != nil {
		fmt.Printf("ANCHOR_TO_ADMIN_REV: Error saving changes: %s", errs)
		return nil, errors.New("Error saving changes")
	}

	bytes, _ := stub.GetState("anchorProgramIDs")

	var anchorProgramIDs Anchor_Program_Holder

	err = json.Unmarshal(bytes, &anchorProgramIDs)
	if err != nil {
		return nil, errors.New("Corrupt Anchor_Program_Holder record")
	}

	anchorProgramIDs.ANCHOR_PROGRAMs = append(anchorProgramIDs.ANCHOR_PROGRAMs, pobox.AnchorProgramID)

	bytes, err = json.Marshal(anchorProgramIDs)
	if err != nil {
		fmt.Print("Error creating Anchor_Program_Holder record")
	}

	err = stub.PutState("anchorProgramIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	return nil, nil // We are Done

}

//=================================================================================================================================
//	 anchor_to_vendor
//=================================================================================================================================
func (t *AssetManagementChaincode) anchor_to_vendor(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if v.AnchorPOAmount == 0 ||
		v.AnchorPoImage == "UNDEFINED" ||
		v.AnchorPoID == "UNDEFINED" ||
		v.VendorID == "UNDEFINED" ||
		v.VendorFName == "UNDEFINED" ||
		v.VendorLName == "UNDEFINED" ||
		v.Vendorphone == "UNDEFINED" ||
		v.Vendoraddress == "UNDEFINED" ||
		v.Vendoremail == "UNDEFINED" ||
		v.Vendorlimit == 0 { //If any part of the order is undefined it has not been fully manufacturered so cannot be sent

		fmt.Printf("ANCHOR_TO_VENDOR: AnchorProgram not fully defined")
		return nil, errors.New("AnchorProgram not fully defined")
	}

	if v.Status == STATE_PROGRAM_INITIATED &&
		v.Owner == string(callerAccount) &&
		caller_affiliation == ROLE_ANCHOR &&
		recipient_affiliation == ROLE_VENDOR &&
		v.Settled == false { // If the roles and users are ok

		v.Owner = receiverAccount              // then make the owner the new owner
		v.Status = STATE_PURCHASE_ORDER_PLACED // and mark it in the state of creating purchase order
		v.PORaisedAgainst = receiverAccount
		//v.POTimestamp = time.Now()
	} else { // Otherwise if there is an error

		fmt.Printf("ANCHOR_TO_VENDOR: Permission Denied")
		return nil, errors.New("Permission Denied")

	}

	_, err := t.save_changes(stub, v) // Write new state

	if err != nil {
		fmt.Printf("ANCHOR_TO_VENDOR: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	return nil, nil // We are Done

}

//=================================================================================================================================
//	 vendor_to_anchor_rev
//=================================================================================================================================
func (t *AssetManagementChaincode) vendor_to_anchor_rev(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string, new_value string) ([]byte, error) {

	var pobox AnchorProgram
	pobox = v

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		v.Owner == string(callerAccount) &&
		caller_affiliation == ROLE_VENDOR &&
		recipient_affiliation == ROLE_ANCHOR &&
		v.Settled == false { // If the roles and users are ok

		pobox.Owner = receiverAccount          // then make the owner the new owner
		pobox.Status = STATE_PROGRAM_INITIATED // and mark it in the state of creating purchase order
		pobox.AnchorProgramID = v.AnchorProgramID + "-R2"
		pobox.AnchorPOAmount = 0 // Update to the new value
		pobox.AnchorPoImage = "UNDEFINED"
		pobox.AnchorPoID = "UNDEFINED"
		pobox.POParent = v.AnchorProgramID
		pobox.PORemarks = new_value
		v.PoForks = append(v.PoForks, pobox.AnchorProgramID)

		record, _ := stub.GetState(pobox.AnchorProgramID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("AnchorProgramID already exists")
		}

	} else { // Otherwise if there is an error

		fmt.Printf("VENDOR_TO_ANCHOR_REV: Permission Denied")
		return nil, errors.New("Permission Denied")

	}

	_, err := t.save_changes(stub, pobox) // Write new state

	if err != nil {
		fmt.Printf("VENDOR_TO_ANCHOR_REV: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	_, errs := t.save_changes(stub, v) // Write new state

	if errs != nil {
		fmt.Printf("VENDOR_TO_ANCHOR_REV: Error saving changes: %s", errs)
		return nil, errors.New("Error saving changes")
	}

	bytes, _ := stub.GetState("anchorProgramIDs")

	var anchorProgramIDs Anchor_Program_Holder

	err = json.Unmarshal(bytes, &anchorProgramIDs)
	if err != nil {
		return nil, errors.New("Corrupt Anchor_Program_Holder record")
	}

	anchorProgramIDs.ANCHOR_PROGRAMs = append(anchorProgramIDs.ANCHOR_PROGRAMs, pobox.AnchorProgramID)

	bytes, err = json.Marshal(anchorProgramIDs)
	if err != nil {
		fmt.Print("Error creating Anchor_Program_Holder record")
	}

	err = stub.PutState("anchorProgramIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	return nil, nil // We are Done

}

//==========================================================================================================
//transfer_vendor_to_anchor_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_vendor_to_anchor_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if x.InvoiceID == "UNDEFINED" ||
		x.InvoiceImage == "UNDEFINED" ||
		x.MOAmount == 0 { //If any part of the order is undefined it has not been fully manufacturered so cannot be sent

		fmt.Printf("VENDOR_TO_ANCHOR_INVOICE: Invoice not fully defined")
		return nil, errors.New("Invoice not fully defined")
	}

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		v.Owner == string(callerAccount) &&
		//v.Settled == false &&
		x.MOStatus == STATE_TEMPLATE &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_VENDOR &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_ANCHOR &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				x.MOStatus = STATE_INVOICE_RAISED // and mark it in the state of creating purchase order
				v.Items[i].MOStatus = STATE_INVOICE_RAISED

				x.InvoiceRaisedAgainst = receiverAccount
				v.Items[i].InvoiceRaisedAgainst = receiverAccount

				//x.MOTimestamp = time.Now()
				//v.Items[i].MOTimestamp = time.Now()

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MoOriginal = f.MOAmount
							v.Items[j].MoOriginal = f.MOAmount

							f.MOAmount = 0
							v.Items[j].MOAmount = 0

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("VENDOR_TO_ANCHOR_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break
			}
		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("VENDOR_TO_ANCHOR_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("VENDOR_TO_ANCHOR_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_rev_anchor_to_vendor_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_rev_anchor_to_vendor_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner == string(callerAccount) &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_RAISED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_ANCHOR &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_VENDOR &&
		v.Settled == false {

		mobox.MOID = x.MOID + "-RIN1"
		mobox.MOOwner = receiverAccount // then make the owner the new owner
		mobox.MOStatus = STATE_TEMPLATE // and mark it in the state of creating purchase order
		mobox.InvoiceID = "UNDEFINED"
		mobox.InvoiceImage = "UNDEFINED"
		mobox.MOAmount = 0
		mobox.MOParent = x.MOID
		mobox.MORemarks = new_value
		x.MOForks = append(x.MOForks, mobox.MOID)

		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("TRANSFER_REV_ANCHOR_TO_VENDOR_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("TRANSFER_REV_ANCHOR_TO_VENDOR_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_REV_ANCHOR_TO_VENDOR_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_anchor_to_vendor_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_anchor_to_vendor_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if x.InvoiceID == "UNDEFINED" ||
		x.InvoiceImage == "UNDEFINED" { //If any part of the order is undefined it has not been fully manufacturered so cannot be sent

		fmt.Printf("ANCHOR_TO_VENDOR_INVOICE: Invoice not fully defined")
		return nil, errors.New("Invoice not fully defined")
	}

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_VENDOR_INVOICE_APPROVED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_ANCHOR &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_VENDOR &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				x.MOStatus = STATE_ANCHOR_AUTHORISED_INVOICE_PAYMENT // and mark it in the state of requesting invoice payment
				v.Items[i].MOStatus = STATE_ANCHOR_AUTHORISED_INVOICE_PAYMENT

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("ANCHOR_TO_VENDOR_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break
			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("ANCHOR_TO_VENDOR_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("ANCHOR_TO_VENDOR_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_rev_vendor_to_anchor_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_rev_vendor_to_anchor_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_ANCHOR_AUTHORISED_INVOICE_PAYMENT &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_VENDOR &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_ANCHOR &&
		v.Settled == false {

		mobox.MOID = x.MOID + "-RIN2"
		mobox.MOOwner = receiverAccount                // then make the owner the new owner
		mobox.MOStatus = STATE_VENDOR_INVOICE_APPROVED // and mark it in the state of creating purchase order

		mobox.ApprovedInvoiceAmount = 0
		mobox.MOParent = x.MOID
		mobox.MORemarks = new_value
		x.MOForks = append(x.MOForks, mobox.MOID)

		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("TRANSFER_REV_VENDOR_TO_ANCHOR_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("TRANSFER_REV_VENDOR_TO_ANCHOR_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_REV_VENDOR_TO_ANCHOR_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_vendor_to_admin_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_vendor_to_admin_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if x.InvoiceID == "UNDEFINED" ||
		x.InvoiceImage == "UNDEFINED" { //If any part of the order is undefined it has not been fully manufacturered so cannot be sent

		fmt.Printf("VENDOR_TO_ADMIN_INVOICE: Invoice not fully defined")
		return nil, errors.New("Invoice not fully defined")
	}

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_ANCHOR_AUTHORISED_INVOICE_PAYMENT &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_VENDOR &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_ADMIN &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				x.MOStatus = STATE_INVOICE_PAYMENT_REQUESTED // and mark it in the state of requesting invoice payment
				v.Items[i].MOStatus = STATE_INVOICE_PAYMENT_REQUESTED

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("VENDOR_TO_ADMIN_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break
			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("VENDOR_TO_ADMIN_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("VENDOR_TO_ADMIN_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_rev_admin_to_vendor_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_rev_admin_to_vendor_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_REQUESTED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_ADMIN &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_VENDOR &&
		v.Settled == false {

		mobox.MOID = x.MOID + "-RIN3"
		mobox.MOOwner = receiverAccount                          // then make the owner the new owner
		mobox.MOStatus = STATE_ANCHOR_AUTHORISED_INVOICE_PAYMENT // and mark it in the state of creating purchase order
		mobox.MOParent = x.MOID
		mobox.MORemarks = new_value
		x.MOForks = append(x.MOForks, mobox.MOID)

		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("TRANSFER_REV_ADMIN_TO_VENDOR_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("TRANSFER_REV_ADMIN_TO_VENDOR_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_REV_ADMIN_TO_VENDOR_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_admin_to_payment_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_admin_to_payment_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if x.InvoiceID == "UNDEFINED" ||
		x.InvoiceImage == "UNDEFINED" { //If any part of the order is undefined it has not been fully manufacturered so cannot be sent

		fmt.Printf("ANCHOR_TO_ADMIN_INVOICE: Invoice not fully defined")
		return nil, errors.New("Invoice not fully defined")
	}

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_REQUESTED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_ADMIN &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_PAYMENT_MAKER &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				x.MOStatus = STATE_INVOICE_PAYMENT_INITIATED // and mark it in the state of requesting invoice payment
				v.Items[i].MOStatus = STATE_INVOICE_PAYMENT_INITIATED

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("TRANSFER_ADMIN_TO_PAYMENT_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break

			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("TRANSFER_ADMIN_TO_PAYMENT_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_ADMIN_TO_PAYMENT_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_rev_payment_to_admin_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_rev_payment_to_admin_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_INITIATED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_MAKER &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_ADMIN &&
		v.Settled == false {

		mobox.MOID = x.MOID + "-RIN4"
		mobox.MOOwner = receiverAccount                  // then make the owner the new owner
		mobox.MOStatus = STATE_INVOICE_PAYMENT_REQUESTED // and mark it in the state of creating purchase order
		mobox.MOParent = x.MOID
		mobox.MORemarks = new_value
		x.MOForks = append(x.MOForks, mobox.MOID)

		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("TRANSFER_REV_PAYMENT_TO_ADMIN_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("TRANSFER_REV_PAYMENT_TO_ADMIN_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_REV_PAYMENT_TO_ADMIN_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_payment_maker_to_payment_checker_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_payment_maker_to_payment_checker_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if x.InvoiceID == "UNDEFINED" ||
		x.InvoiceImage == "UNDEFINED" { //If any part of the order is undefined it has not been fully manufacturered so cannot be sent

		fmt.Printf("TRANSFER PAYMENT MAKER TO PAYMENT CHECKER_INVOICE: Invoice not fully defined")
		return nil, errors.New("Invoice not fully defined")
	}

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_INITIATED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_MAKER &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_PAYMENT_CHECKER &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				x.MOStatus = STATE_INVOICE_PAYMENT_PENDING_APPROVAL // and mark it in the state of requesting invoice payment
				v.Items[i].MOStatus = STATE_INVOICE_PAYMENT_PENDING_APPROVAL

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("TRANSFER_PAYMENT_MAKER_TO_PAYMENT_CHECKER_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break

			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("TRANSFER_PAYMENT_MAKER_TO_PAYMENT_CHECKER_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_PAYMENT_MAKER_TO_PAYMENT_CHECKER_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_rev_payment_checker_to_payment_maker_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_rev_payment_checker_to_payment_maker_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_PENDING_APPROVAL &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_PAYMENT_MAKER &&
		v.Settled == false {

		mobox.MOID = x.MOID + "-RIN5"
		mobox.MOOwner = receiverAccount                  // then make the owner the new owner
		mobox.MOStatus = STATE_INVOICE_PAYMENT_INITIATED // and mark it in the state of creating purchase order
		mobox.MOParent = x.MOID
		x.MOForks = append(x.MOForks, mobox.MOID)
		mobox.MORemarks = new_value
		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("TRANSFER_REV_PAYMENT_CHECKER_TO_PAYMENT_MAKER_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("TRANSFER_REV_PAYMENT_CHECKER_TO_PAYMENT_MAKER_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_REV_PAYMENT_CHECKER_TO_PAYMENT_MAKER_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_payment_checker_to_payment_maker_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_payment_checker_to_payment_maker_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_PENDING_APPROVAL &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_PAYMENT_MAKER &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				x.MOStatus = STATE_INVOICE_PAYMENT_INITIATED // and mark it in the state of requesting invoice payment
				v.Items[i].MOStatus = STATE_INVOICE_PAYMENT_INITIATED

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("TRANSFER_PAYMENT_CHECKER_TO_PAYMENT_MAKER_INVOICE: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break

			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("TRANSFER_PAYMENT_CHECKER_TO_PAYMENT_MAKER_INVOICE: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("TRANSFER_PAYMENT_CHECKER_TO_PAYMENT_MAKER_INVOICE: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

/*//==========================================================================================================
//transfer_payment_checker_to_anchor_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_payment_checker_to_anchor_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAID &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == true &&
		x.AnchorPaidInv == false &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_ANCHOR &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				break

			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("transfer_payment_checker_to_anchor_invoice: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("transfer_payment_checker_to_anchor_invoice: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//==========================================================================================================
//transfer_anchor_to_payment_checker_invoice
//==========================================================================================================
func (t *AssetManagementChaincode) transfer_anchor_to_payment_checker_invoice(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, receiverAccount string, recipient_affiliation string) ([]byte, error) {

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		//v.Owner 							 == caller &&
		//v.Settled == false &&
		x.MOStatus == STATE_ANCHOR_PAID_INVOICE &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_ANCHOR &&
		x.MOPaid == true &&
		x.AnchorPaidInv == true &&
		x.MOSettled == false &&
		recipient_affiliation == ROLE_PAYMENT_CHECKER &&
		v.Settled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOOwner = receiverAccount // then make the owner the new owner
				v.Items[i].MOOwner = receiverAccount

				break

			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("transfer_anchor_to_payment_checker_invoice: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("transfer_anchor_to_payment_checker_invoice: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

*/
//=================================================================================================================================
//	 Update Functions
//---------------------------------------------------------------------------------------------------------------------------------
//   ADMIN UPDATE ANCHOR FUNCTIONS
//=================================================================================================================================
//	 update_anchor_details
//=================================================================================================================================
func (t *AssetManagementChaincode) update_anchor_details(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation, name, id, ifsc, agreement, account, limit, expiry, interest, graceInterest, graceinterestPeriod, penalInterest, anchorLiquidation string) ([]byte, error) {
	new_amount, _ := strconv.ParseFloat(string(limit), 64) // will return an error if the new purchase amount contains non numerical chars

	if v.Status == STATE_TEMPLATE &&
		v.Owner == string(callerAccount) &&
		caller_affiliation == ROLE_ADMIN &&
		v.Settled == false {

		v.AnchorName = name
		v.AnchorID = id
		v.AnchorIFSCCode = ifsc
		v.AnchorAgreement = agreement
		v.AnchorAccountNo = account
		v.AnchorLimit = new_amount
		v.AnchorExpiryDate = expiry
		v.AnchorInterest = interest
		v.AnchorGarceInterest = graceInterest
		v.AnchorGarceInterestperiod = graceinterestPeriod
		v.AnchorPenalInterest = penalInterest
		v.AnchorLiquidation = anchorLiquidation

	} else {

		return nil, errors.New("Permission denied")

	}

	_, err := t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("update_anchor_details: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	return nil, nil

}

//---------------------------------------------------------------------------------------------------------------------------------
//   ADMIN UPDATE VENDOR FUNCTIONS
//=================================================================================================================================
//	 update_vendor_details
//=================================================================================================================================
func (t *AssetManagementChaincode) update_vendor_details(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation, id, limit, firstName, lastName, email, phone, address, pan, agreement, expiry, bank, bankAddress, account, ifsc string) ([]byte, error) {
	new_amount, _ := strconv.ParseFloat(string(limit), 64) // will return an error if the new purchase amount contains non numerical chars

	if v.Status == STATE_TEMPLATE &&
		v.Owner == string(callerAccount) &&
		caller_affiliation == ROLE_ADMIN &&
		v.Settled == false {

		v.VendorID = id
		v.Vendorlimit = new_amount
		v.VendorFName = firstName
		v.VendorLName = lastName
		v.Vendoremail = email
		v.Vendorphone = phone
		v.Vendoraddress = address
		v.Vendorpanno = pan
		v.VendorAgreement = agreement
		v.VendorExpirydate = expiry
		v.Vendorbank = bank
		v.Vendorbaddress = bankAddress
		v.Vendoraccountno = account
		v.Vendorifsccode = ifsc

	} else {

		return nil, errors.New("Permission denied")

	}

	_, err := t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("update_vendor_details: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	return nil, nil

}

//---------------------------------------------------------------------------------------------------------------------------------
//   ANCHOR UPDATE PO FUNCTIONS
//=================================================================================================================================
//	 update_anchor_purchase_order
//=================================================================================================================================

func (t *AssetManagementChaincode) update_anchor_purchase_order(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value, poImage, poID string) ([]byte, error) {

	new_amount, _ := strconv.ParseFloat(string(new_value), 64) // will return an error if the new purchase amount contains non numerical chars

	if new_amount > v.Vendorlimit {
		fmt.Println("Amount exceeds authorized vendor limit")
		return nil, nil
	} else if v.Status == STATE_PROGRAM_INITIATED &&
		v.Owner == string(callerAccount) &&
		caller_affiliation == ROLE_ANCHOR &&
		v.AnchorPOAmount == 0 && // Can't change the purchase amount after its initial assignment
		v.Settled == false {

		v.AnchorPOAmount = new_amount // Update to the new value
		v.AnchorPoImage = poImage
		v.AnchorPoID = poID

	} else {

		return nil, errors.New("Permission denied")

	}

	_, err := t.save_changes(stub, v) // Save the changes in the blockchain

	if err != nil {
		fmt.Printf("update_anchor_purchase_order: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	return nil, nil

}

//---------------------------------------------------------------------------------------------------------------------------------
//   VENDOR UPDATE PO FUNCTIONS
//=================================================================================================================================
//	 update_vendor_po_acknowledgement
//=================================================================================================================================
func (t *AssetManagementChaincode) update_vendor_po_acknowledgement(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		v.Owner == string(callerAccount) &&
		caller_affiliation == ROLE_VENDOR &&
		v.POAcknowledged == false &&
		v.Settled == false {

		v.POAcknowledged = true

	} else {
		return nil, errors.New("Permission denied")
	}

	_, err := t.save_changes(stub, v)
	if err != nil {
		fmt.Printf("UPDATE_VENDOR_PO_ACKNOWLEDGEMENT: Error saving changes: %s", err)
		return nil, errors.New("UPDATE_VENDOR_PO_ACKNOWLEDGEMENT: Error saving changes")
	}

	return nil, nil

}

//---------------------------------------------------------------------------------------------------------------------------------
//   VENDOR UPDATE INVOICE FUNCTIONS
//=================================================================================================================================
//	 update_vendor_invoice_details
//=================================================================================================================================
func (t *AssetManagementChaincode) update_vendor_invoice_details(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation, new_value, invID, image string) ([]byte, error) {
	new_amount, _ := strconv.ParseFloat(string(new_value), 64) // will return an error if the new purchase amount contains non numerical chars

	var inv float64
	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		v.Owner == string(callerAccount) &&
		v.Settled == false &&
		x.MOStatus == STATE_TEMPLATE &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_VENDOR &&
		new_amount <= v.AnchorPOAmount &&
		x.MOPaid == false &&
		x.MOSettled == false {

		for i := range v.Items {
			if v.Items[i].MOStatus == STATE_VENDOR_INVOICE_APPROVED {
				inv += v.Items[i].ApprovedInvoiceAmount
				continue
			}
		}

		if inv+new_amount > v.AnchorPOAmount {
			fmt.Println("Total invoice amount cannot exceed the Purchase Order")
			return nil, nil
		} else {

			for i := range v.Items {
				if x.MOID == v.Items[i].MOID {

					x.MOAmount = new_amount
					v.Items[i].MOAmount = new_amount

					x.InvoiceID = invID
					v.Items[i].InvoiceID = invID

					x.InvoiceImage = image
					v.Items[i].InvoiceImage = image

					break

				}

			}
		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("UPDATE_VENDOR_INVOICE_DETAILS: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_VENDOR_INVOICE_DETAILS: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//---------------------------------------------------------------------------------------------------------------------------------
//   ANCHOR UPDATE INVOICE FUNCTIONS
//=================================================================================================================================
//	 update_anchor_invoice_authorized_amount
//=================================================================================================================================
func (t *AssetManagementChaincode) update_anchor_invoice_authorized_amount(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value string) ([]byte, error) {
	new_amount, _ := strconv.ParseFloat(string(new_value), 64) // will return an error if the new purchase amount contains non numerical chars

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_RAISED ||
		x.MOStatus == STATE_VENDOR_INVOICE_APPROVED &&
			x.MOOwner == string(callerAccount) &&
			caller_affiliation == ROLE_ANCHOR &&
			x.MOPaid == false &&
			x.MOSettled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.ApprovedInvoiceAmount = new_amount
				v.Items[i].ApprovedInvoiceAmount = new_amount

				x.MOStatus = STATE_VENDOR_INVOICE_APPROVED
				v.Items[i].MOStatus = STATE_VENDOR_INVOICE_APPROVED

				break
			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("UPDATE_ANCHOR_AUTHORIZED_INVOICE_AMOUNT: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_ANCHOR_AUTHORIZED_INVOICE_AMOUNT: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

/*//=================================================================================================================================
//	 update_rev_anchor_invoice_authorized_amount
//=================================================================================================================================
func (t *AssetManagementChaincode) update_rev_anchor_invoice_authorized_amount(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value string) ([]byte, error) {
	new_amount, _ := strconv.ParseFloat(string(new_value), 64) // will return an error if the new purchase amount contains non numerical chars

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_RAISED ||
		x.MOStatus == STATE_VENDOR_INVOICE_APPROVED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_ANCHOR &&
		x.MOPaid == false &&
		x.MOSettled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.ApprovedInvoiceAmount = new_amount
				v.Items[i].ApprovedInvoiceAmount = new_amount

				x.MOStatus = STATE_VENDOR_INVOICE_APPROVED
				v.Items[i].MOStatus = STATE_VENDOR_INVOICE_APPROVED

				break
			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("UPDATE_ANCHOR_AUTHORIZED_INVOICE_AMOUNT: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_ANCHOR_AUTHORIZED_INVOICE_AMOUNT: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}
*/

//---------------------------------------------------------------------------------------------------------------------------------
//   PAYMENT MAKER UPDATE INVOICE FUNCTIONS
//=================================================================================================================================
//	 update_maker_invoice_payment
//=================================================================================================================================
func (t *AssetManagementChaincode) update_maker_invoice_payment(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value, channel string) ([]byte, error) {
	new_amount, _ := strconv.ParseFloat(string(new_value), 64) // will return an error if the new purchase amount contains non numerical chars

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_INITIATED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_MAKER &&
		x.MOPaid == false &&
		x.MOSettled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOReceivableAmount = new_amount
				v.Items[i].MOReceivableAmount = new_amount

				x.PaymentChannel = channel
				v.Items[i].PaymentChannel = channel

				break
			}

		}
	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("UPDATE_MAKER_INVOICE_PAYMENT: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_MAKER_INVOICE_PAYMENT: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//---------------------------------------------------------------------------------------------------------------------------------
//   PAYMENT CHECKER UPDATE INVOICE FUNCTIONS
//=================================================================================================================================
//	 update_checker_invoice_approval
//=================================================================================================================================
func (t *AssetManagementChaincode) update_checker_invoice_approval(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	//var pending = 0
	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_PENDING_APPROVAL &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == false &&
		x.MOSettled == false {

		for i := range v.Items {
			//			pending += v.Items[i].MOAmount
			if x.MOID == v.Items[i].MOID {

				x.CheckerApprovedPayment = true
				v.Items[i].CheckerApprovedPayment = true

				x.MOStatus = STATE_INVOICE_PAYMENT_APPROVED
				v.Items[i].MOStatus = STATE_INVOICE_PAYMENT_APPROVED

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("UPDATE_CHECKER_INVOICE_APPROVAL: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break

			}

		}

	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("UPDATE_CHECKER_INVOICE_APPROVAL: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_CHECKER_INVOICE_APPROVAL: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//=================================================================================================================================
//	 update_rev_checker_invoice_approval
//=================================================================================================================================
func (t *AssetManagementChaincode) update_rev_checker_invoice_approval(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	//var pending = 0
	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_APPROVED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == false &&
		x.MOSettled == false {

		mobox.MOID = x.MOID + "-RIU1"
		mobox.CheckerApprovedPayment = false                    // then make the owner the new owner
		mobox.MOStatus = STATE_INVOICE_PAYMENT_PENDING_APPROVAL // and mark it in the state of creating purchase order
		mobox.MOParent = x.MOID
		mobox.MORemarks = new_value
		x.MOForks = append(x.MOForks, mobox.MOID)

		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("UPDATE_REV_PAYMENT_CHECKER_INVOICE_APPROVAL: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("UPDATE_REV_PAYMENT_CHECKER_INVOICE_APPROVAL: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_REV_PAYMENT_CHECKER_INVOICE_APPROVAL: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//=================================================================================================================================
//	 update_checker_invoice_payment
//=================================================================================================================================
func (t *AssetManagementChaincode) update_checker_invoice_payment(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value0 string, new_value string) ([]byte, error) {

	//var pending = 0
	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAYMENT_APPROVED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == false &&
		x.MOSettled == false {

		for i := range v.Items {
			//			pending += v.Items[i].MOAmount
			if x.MOID == v.Items[i].MOID {
				
				x.TxnStatus = new_value0
				v.Items[i].TxnStatus = new_value0 
				
				if new_value0 == "SUCCESS" {
					
					x.UTRNumber = new_value
					v.Items[i].UTRNumber = new_value

					x.MOPaid = true
					v.Items[i].MOPaid = true

					x.MOStatus = STATE_INVOICE_PAID
					v.Items[i].MOStatus = STATE_INVOICE_PAID
				} else {
					
					
					x.UTRNumber = new_value
					v.Items[i].UTRNumber = new_value

					x.MOPaid = false
					v.Items[i].MOPaid = false

					x.MOStatus = STATE_INVOICE_PAYMENT_APPROVED
					v.Items[i].MOStatus = STATE_INVOICE_PAYMENT_APPROVED
				}

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("UPDATE_CHECKER_INVOICE_PAYMENT: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break

			}

		}

	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("UPDATE_CHECKER_INVOICE_PAYMENT: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_CHECKER_INVOICE_PAYMENT: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//=================================================================================================================================
//	 update_rev_checker_invoice_payment
//=================================================================================================================================
func (t *AssetManagementChaincode) update_rev_checker_invoice_payment(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	//var pending = 0
	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAID &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == true &&
		x.MOSettled == false {

		mobox.MOID = x.MOID + "-RIU2"
		mobox.MOPaid = false // then make the owner the new owner
		mobox.UTRNumber = "UNDEFINED"
		mobox.MOStatus = STATE_INVOICE_PAYMENT_APPROVED // and mark it in the state of creating purchase order
		mobox.MOParent = x.MOID
		mobox.MORemarks = new_value
		x.MOForks = append(x.MOForks, mobox.MOID)

		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("UPDATE_REV_CHECKER_INVOICE_PAYMENT: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("UPDATE_REV_CHECKER_INVOICE_PAYMENT: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_REV_CHECKER_INVOICE_PAYMENT: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//=================================================================================================================================
//	 update_checker_invoice_settlement
//=================================================================================================================================
func (t *AssetManagementChaincode) update_checker_invoice_settlement(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value string) ([]byte, error) {

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_PAID &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == true &&
		x.MOSettled == false {

		for i := range v.Items {
			if x.MOID == v.Items[i].MOID {

				x.MOSettled = true
				v.Items[i].MOSettled = true

				x.MOStatus = STATE_INVOICE_SETTLED
				v.Items[i].MOStatus = STATE_INVOICE_SETTLED

				x.SettlementAmount = new_value
				v.Items[i].SettlementAmount = new_value

				if x.MOParent == "" || x.MOParent == "UNDEFINED" {
					fmt.Println("QUERY: Error retrieving Parent")
				} else {
					f, err := t.retrieve_invoice(stub, x.MOParent)
					if err != nil {
						fmt.Printf("QUERY: Error retrieving invoice: %s", err)
						return nil, errors.New("QUERY: Error retrieving invoice " + err.Error())
					}

					for j := range v.Items {
						if f.MOID == v.Items[j].MOID {

							f.MOStatus = STATE_INVOICE_RETIRED
							v.Items[j].MOStatus = STATE_INVOICE_RETIRED

							break
						}

					}

					_, errs := t.save_invoice(stub, f)

					if errs != nil {
						fmt.Printf("UPDATE_CHECKER_INVOICE_SETTLEMENT: $!^!* Error saving changes to Invoice: %s", errs)
						return nil, errors.New(" %@((@& Error saving changes to invoice")
					}

				}

				break

			}

		}

	}

	_, err := t.save_invoice(stub, x)

	if err != nil {
		fmt.Printf("UPDATE_CHECKER_INVOICE_SETTLEMENT: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_CHECKER_INVOICE_SETTLEMENT: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//=================================================================================================================================
//	 update_rev_checker_invoice_settlement
//=================================================================================================================================
func (t *AssetManagementChaincode) update_rev_checker_invoice_settlement(stub shim.ChaincodeStubInterface, x MyBoxItem, v AnchorProgram, callerAccount []byte, caller_affiliation string, new_value string) ([]byte, error) {

	var mobox MyBoxItem
	mobox = x

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller &&
		v.Settled == false &&
		x.MOStatus == STATE_INVOICE_SETTLED &&
		x.MOOwner == string(callerAccount) &&
		caller_affiliation == ROLE_PAYMENT_CHECKER &&
		x.MOPaid == true &&
		x.MOSettled == true {

		mobox.MOID = x.MOID + "-RIU3"
		mobox.MOSettled = false // then make the owner the new owner
		mobox.SettlementAmount = "UNDEFINED"
		mobox.MOStatus = STATE_INVOICE_PAID // and mark it in the state of creating purchase order
		mobox.MOParent = x.MOID
		mobox.MORemarks = new_value
		x.MOForks = append(x.MOForks, mobox.MOID)

		record, _ := stub.GetState(mobox.MOID) // If not an error then a record exists so cant create a new order with this AnchorProgramID as it must be unique
		if record != nil {
			return nil, errors.New("Invoice already exists")
		}

	}

	_, err := t.save_invoice(stub, mobox)

	if err != nil {
		fmt.Printf("UPDATE_REV_CHECKER_INVOICE_SETTLEMENT: $!^!* Error saving changes to Invoice: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder record")
	}

	invoiceIDs.INVOICEs = append(invoiceIDs.INVOICEs, mobox.MOID)

	bytes, err = json.Marshal(invoiceIDs)
	if err != nil {
		fmt.Print("Error creating Invoice_Holder record")
	}

	err = stub.PutState("invoiceIDs", bytes)
	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	v.Items = append(v.Items, (mobox))

	_, errs := t.save_invoice(stub, x)

	if errs != nil {
		fmt.Printf("UPDATE_REV_CHECKER_INVOICE_SETTLEMENT: $!^!* Error saving changes to Invoice: %s", errs)
		return nil, errors.New(" %@((@& Error saving changes to invoice")
	}

	_, err = t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("UPDATE_REV_CHECKER_INVOICE_SETTLEMENT: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//=================================================================================================================================
//	 settlement_anchorprogram
//=================================================================================================================================
func (t *AssetManagementChaincode) settlement_anchorprogram(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	if v.Status == STATE_PURCHASE_ORDER_PLACED &&
		// v.Owner 						== caller                      &&
		v.Settled == false &&
		caller_affiliation == ROLE_PAYMENT_CHECKER {

		v.Settled = true
		v.Status = STATE_ANCHOR_PROGRAM_CLOSED

	}

	_, err := t.save_changes(stub, v)

	if err != nil {
		fmt.Printf("settlement_anchorprogram: $!^!* Error saving changes to AnchorProgram: %s", err)
		return nil, errors.New(" %@((@& Error saving changes to AnchorProgram")
	}

	return nil, nil

}

//=================================================================================================================================
//	Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//  		initial arguments passed are passed on to the called function.
//=================================================================================================================================
func (t *AssetManagementChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	callerRole, err := stub.ReadCertAttribute("role")
	if err != nil {
		fmt.Printf("Error reading attribute 'role' [%v] \n", err)
		return nil, fmt.Errorf("Failed fetching caller role. Error was [%v]", err)
	}
	caller_affiliation := string(callerRole[:])

	callerAccount, err := stub.ReadCertAttribute("account")
	if err != nil {
		return nil, fmt.Errorf("Failed fetching caller account. Error was [%v]", err)
	}

	if function == "get_anchorprogram_details" {
		if len(args) != 1 {
			fmt.Printf("Incorrect number of arguments passed")
			return nil, errors.New("QUERY: Incorrect number of arguments passed")
		}

		v, err := t.retrieve_anchorprogram(stub, args[0])
		if err != nil {
			fmt.Printf("QUERY: Error retrieving po: %s", err)
			return nil, errors.New("QUERY: Error retrieving anchor program " + err.Error())
		}

		return t.get_anchorprogram_details(stub, v, callerAccount, caller_affiliation)

	} else if function == "get_invoice_details" {
		if len(args) != 1 {
			fmt.Printf("Incorrect number of arguments passed")
			return nil, errors.New("QUERY: Incorrect number of arguments passed")
		}

		x, errs := t.retrieve_invoice(stub, args[0])
		if errs != nil {
			fmt.Printf("QUERY: Error retrieving invoice: %s", errs)
			return nil, errors.New("QUERY: Error retrieving invoice " + errs.Error())
		}

		return t.get_invoice_details(stub, x, callerAccount, caller_affiliation)

	} else if function == "get_anchorprograms" {
		return t.get_anchorprograms(stub, callerAccount, caller_affiliation)
	} else if function == "get_anchorprogramIDs" {
		return t.get_anchorprogramIDs(stub, callerAccount, caller_affiliation)
	} else if function == "get_invoiceIDs" {
		return t.get_invoiceIDs(stub, callerAccount, caller_affiliation)
	} else if function == "get_invoices" {
		return t.get_invoices(stub, callerAccount, caller_affiliation)
	}

	return nil, errors.New("Received unknown function invocation")

}

//=================================================================================================================================
//	 Read Functions
//=================================================================================================================================
//	 get_invoice_details -----> get transaction details for a particular order invoice
//=================================================================================================================================
func (t *AssetManagementChaincode) get_invoice_details(stub shim.ChaincodeStubInterface, v MyBoxItem, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	bytes, err := json.Marshal(v)

	if err != nil {
		return nil, errors.New("GET_INVOICE_DETAILS: Invalid Invoice object")
	}

	if v.MOOwner == string(callerAccount) ||
		v.InvoiceRaisedBy == string(callerAccount) ||
		caller_affiliation == ROLE_ADMIN {

		return bytes, nil
	} else {
		return nil, errors.New("Permission Denied")
	}

}

//=================================================================================================================================
//	 get_anchorprogram_details -----> get transaction details for a particular order
//=================================================================================================================================
func (t *AssetManagementChaincode) get_anchorprogram_details(stub shim.ChaincodeStubInterface, v AnchorProgram, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	bytes, err := json.Marshal(v)

	if err != nil {
		return nil, errors.New("GET_ANCHORPROGRAM_DETAILS: Invalid AnchorProgram object")
	}

	if v.Owner == string(callerAccount) ||
		v.POraisedBy == string(callerAccount) ||
		caller_affiliation == ROLE_ADMIN {

		return bytes, nil
	} else {
		return nil, errors.New("Permission Denied")
	}

}

//=================================================================================================================================
//	 get_invoices ----> get details of all invoices
//=================================================================================================================================

func (t *AssetManagementChaincode) get_invoices(stub shim.ChaincodeStubInterface, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder")
	}

	result := "["

	var temp []byte
	var v MyBoxItem

	for _, po := range invoiceIDs.INVOICEs {

		v, err = t.retrieve_invoice(stub, po)
		if err != nil {
			return nil, errors.New("Failed to retrieve Invoice")
		}

		temp, err = t.get_invoice_details(stub, v, callerAccount, caller_affiliation)
		if err == nil {
			result += string(temp) + ","
		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

//=================================================================================================================================
//	 get_anchorprograms ----> get details of all orders
//=================================================================================================================================

func (t *AssetManagementChaincode) get_anchorprograms(stub shim.ChaincodeStubInterface, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	bytes, err := stub.GetState("anchorProgramIDs")
	if err != nil {
		return nil, errors.New("Unable to get anchorProgramIDs")
	}

	var anchorProgramIDs Anchor_Program_Holder

	err = json.Unmarshal(bytes, &anchorProgramIDs)
	if err != nil {
		return nil, errors.New("Corrupt Anchor_Program_Holder")
	}

	result := "["

	var temp []byte
	var v AnchorProgram

	for _, po := range anchorProgramIDs.ANCHOR_PROGRAMs {

		v, err = t.retrieve_anchorprogram(stub, po)
		if err != nil {
			return nil, errors.New("Failed to retrieve Anchor Program")
		}

		temp, err = t.get_anchorprogram_details(stub, v, callerAccount, caller_affiliation)
		if err == nil {
			result += string(temp) + ","
		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

//=================================================================================================================================
//	 get_anchorprogramIDs ----> get ID details of all orders
//=================================================================================================================================

func (t *AssetManagementChaincode) get_anchorprogramIDs(stub shim.ChaincodeStubInterface, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	bytes, err := stub.GetState("anchorProgramIDs")
	if err != nil {
		return nil, errors.New("Unable to get anchorProgramIDs")
	}

	var anchorProgramIDs Anchor_Program_Holder

	err = json.Unmarshal(bytes, &anchorProgramIDs)
	if err != nil {
		return nil, errors.New("Corrupt Anchor_Program_Holder")
	}

	result := "["

	var v AnchorProgram
	var list ProgramIDs

	for _, po := range anchorProgramIDs.ANCHOR_PROGRAMs {

		v, err = t.retrieve_anchorprogram(stub, po)
		if err != nil {
			return nil, errors.New("Failed to retrieve Anchor Program")
		}

		if v.Owner == string(callerAccount) ||
			v.POraisedBy == string(callerAccount) ||
			v.PORaisedAgainst == string(callerAccount) ||
			caller_affiliation == ROLE_ADMIN {

			list.AnchorID = v.AnchorID
			list.POraisedBy = v.POraisedBy
			list.AnchorName = v.AnchorName
			list.AnchorAccountNo = v.AnchorAccountNo
			list.AnchorPOAmount = v.AnchorPOAmount
			list.AnchorIFSCCode = v.AnchorIFSCCode
			list.AnchorLimit = v.AnchorLimit
			list.AnchorExpiryDate = v.AnchorExpiryDate
			list.AnchorInterest = v.AnchorInterest
			list.AnchorGarceInterest = v.AnchorGarceInterest
			list.AnchorGarceInterestperiod = v.AnchorGarceInterestperiod
			list.AnchorPenalInterest = v.AnchorPenalInterest
			list.AnchorLiquidation = v.AnchorLiquidation
			list.AnchorPoID = v.AnchorPoID
			list.VendorID = v.VendorID
			list.VendorFName = v.VendorFName
			list.VendorLName = v.VendorLName
			list.Vendoremail = v.Vendoremail
			list.Vendorphone = v.Vendorphone
			list.Vendorpanno = v.Vendorpanno
			list.Vendoraddress = v.Vendoraddress
			list.Vendorbank = v.Vendorbank
			list.Vendorbaddress = v.Vendorbaddress
			list.Vendoraccountno = v.Vendoraccountno
			list.Vendorifsccode = v.Vendorifsccode
			list.Vendorlimit = v.Vendorlimit
			list.VendorExpirydate = v.VendorExpirydate
			list.POAcknowledged = v.POAcknowledged
			//list.POTimestamp = v.POTimestamp
			list.Status = v.Status
			list.AnchorProgramID = v.AnchorProgramID
			list.Settled = v.Settled

			temp, err := json.Marshal(list)

			if err == nil {
				result += string(temp) + ","
			}
		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

//=================================================================================================================================
//	 get_invoiceIDs ----> get ID details of all orders
//=================================================================================================================================

func (t *AssetManagementChaincode) get_invoiceIDs(stub shim.ChaincodeStubInterface, callerAccount []byte, caller_affiliation string) ([]byte, error) {

	bytes, err := stub.GetState("invoiceIDs")
	if err != nil {
		return nil, errors.New("Unable to get invoiceIDs")
	}

	var invoiceIDs Invoice_Holder

	err = json.Unmarshal(bytes, &invoiceIDs)
	if err != nil {
		return nil, errors.New("Corrupt Invoice_Holder")
	}

	result := "["

	var v MyBoxItem
	var list InvoiceIDs

	for _, po := range invoiceIDs.INVOICEs {

		v, err = t.retrieve_invoice(stub, po)
		if err != nil {
			return nil, errors.New("Failed to retrieve Invoice")
		}

		if v.MOOwner == string(callerAccount) ||
			v.InvoiceRaisedBy == string(callerAccount) ||
			v.InvoiceRaisedAgainst == string(callerAccount) ||
			caller_affiliation == ROLE_ADMIN {

			list.POID = v.POID
			list.MOID = v.MOID
			list.MOOwner = v.MOOwner
			list.AnchorName = v.AnchorName
			list.AnchorAccountNo = v.AnchorAccountNo
			list.AnchorPOAmount = v.AnchorPOAmount
			list.AnchorIFSCCode = v.AnchorIFSCCode
			list.AnchorInterest = v.AnchorInterest
			list.InvoiceRaisedBy = v.InvoiceRaisedBy
			list.InvoiceRaisedAgainst = v.InvoiceRaisedAgainst
			list.MOAmount = v.MOAmount
			list.InvoiceID = v.InvoiceID
			list.ApprovedInvoiceAmount = v.ApprovedInvoiceAmount
			list.MOStatus = v.MOStatus
			list.CheckerApprovedPayment = v.CheckerApprovedPayment
			//list.MOTimestamp = v.MOTimestamp
			list.MOPaid = v.MOPaid
			list.UTRNumber = v.UTRNumber
			list.MOSettled = v.MOSettled
			list.Vendorfname = v.Vendorfname
			list.Vendorbank = v.Vendorbank
			list.Vendorifsccode = v.Vendorifsccode
			list.AnchorPoID = v.AnchorPoID

			temp, err := json.Marshal(list)

			if err == nil {
				result += string(temp) + ","
			}
		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

//=================================================================================================================================
//	 Main - main - Starts up the chaincode
//=================================================================================================================================
func main() {
	err := shim.Start(new(AssetManagementChaincode))
	if err != nil {
		fmt.Printf("Error starting AssetManagementChaincode: %s", err)
	}
}
