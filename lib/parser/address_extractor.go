package parser

import (
	"encoding/json"
	"strings"

	"github.com/1119-Labs/callisto/v4/lib/types"
)

// AddressExtractor extracts account addresses from transaction messages.
// It uses a registry of message-type-specific extractors with a fallback to generic extraction.
type AddressExtractor struct {
	// Registry of message type -> extractor function
	extractors map[string]MessageAddressExtractor
}

// MessageAddressExtractor is a function that extracts addresses from a specific message type.
type MessageAddressExtractor func(msgBytes json.RawMessage) []string

// NewAddressExtractor creates a new AddressExtractor with registered extractors.
func NewAddressExtractor() *AddressExtractor {
	ae := &AddressExtractor{
		extractors: make(map[string]MessageAddressExtractor),
	}

	// Register specific message type extractors
	ae.registerExtractors()

	return ae
}

// registerExtractors registers all known message type extractors.
func (ae *AddressExtractor) registerExtractors() {
	// Perpx specific messages
	ae.extractors["/perpx.sending.MsgCreateTransfer"] = extractPerpxTransfer
	ae.extractors["/perpx.sending.MsgWithdrawFromSubaccount"] = extractPerpxSubaccountMsg
	ae.extractors["/perpx.sending.MsgDepositToSubaccount"] = extractPerpxSubaccountMsg
	ae.extractors["/perpx.clob.MsgPlaceOrder"] = extractPerpxOrderMsg
	ae.extractors["/perpx.clob.MsgCancelOrder"] = extractPerpxOrderMsg
	ae.extractors["/perpx.clob.MsgProposedOperations"] = extractPerpxProposedOperations

	// Standard Cosmos SDK messages
	ae.extractors["/cosmos.bank.v1beta1.MsgSend"] = extractBankSend
	ae.extractors["/cosmos.bank.v1beta1.MsgMultiSend"] = extractBankMultiSend
	ae.extractors["/cosmos.staking.v1beta1.MsgDelegate"] = extractStakingDelegate
	ae.extractors["/cosmos.staking.v1beta1.MsgUndelegate"] = extractStakingDelegate
	ae.extractors["/cosmos.staking.v1beta1.MsgBeginRedelegate"] = extractStakingRedelegate
	ae.extractors["/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward"] = extractDistributionWithdraw
	ae.extractors["/cosmos.gov.v1beta1.MsgVote"] = extractGovVote
	ae.extractors["/cosmos.gov.v1.MsgVote"] = extractGovVote
	ae.extractors["/cosmos.gov.v1beta1.MsgDeposit"] = extractGovDeposit
	ae.extractors["/cosmos.gov.v1.MsgDeposit"] = extractGovDeposit
	ae.extractors["/cosmos.gov.v1beta1.MsgSubmitProposal"] = extractGovProposal
	ae.extractors["/cosmos.gov.v1.MsgSubmitProposal"] = extractGovProposal
	ae.extractors["/cosmos.authz.v1beta1.MsgGrant"] = extractAuthzGrant
	ae.extractors["/cosmos.authz.v1beta1.MsgRevoke"] = extractAuthzGrant
	ae.extractors["/cosmos.authz.v1beta1.MsgExec"] = extractAuthzExec
	ae.extractors["/cosmos.feegrant.v1beta1.MsgGrantAllowance"] = extractFeeGrant
	ae.extractors["/cosmos.feegrant.v1beta1.MsgRevokeAllowance"] = extractFeeGrant

	// IBC messages
	ae.extractors["/ibc.applications.transfer.v1.MsgTransfer"] = extractIBCTransfer
}

// ExtractFromTx extracts all unique account addresses from a transaction.
func (ae *AddressExtractor) ExtractFromTx(tx *types.Transaction) []string {
	uniqueAddresses := make(map[string]struct{})

	// Check for nil tx or tx body
	if tx == nil || tx.Tx == nil || tx.Tx.Body == nil {
		return []string{}
	}

	for _, msg := range tx.Tx.Body.Messages {
		addresses := ae.ExtractFromMessage(msg)
		for _, addr := range addresses {
			if addr != "" {
				uniqueAddresses[addr] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(uniqueAddresses))
	for addr := range uniqueAddresses {
		result = append(result, addr)
	}

	return result
}

// ExtractFromMessage extracts addresses from a single message.
func (ae *AddressExtractor) ExtractFromMessage(msg types.Message) []string {
	msgType := msg.GetType()
	msgBytes := msg.GetBytes()

	// Try specific extractor first
	if extractor, ok := ae.extractors[msgType]; ok {
		return extractor(msgBytes)
	}

	// Fallback to generic extraction
	return genericExtractor(msgBytes)
}

// =============================================================================
// Perpx-specific extractors
// =============================================================================

// extractPerpxTransfer handles /perpx.sending.MsgCreateTransfer
// Structure: { "transfer": { "sender": { "owner": "..." }, "recipient": { "owner": "..." } } }
func extractPerpxTransfer(msgBytes json.RawMessage) []string {
	var msg struct {
		Transfer struct {
			Sender struct {
				Owner string `json:"owner"`
			} `json:"sender"`
			Recipient struct {
				Owner string `json:"owner"`
			} `json:"recipient"`
		} `json:"transfer"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.Transfer.Sender.Owner != "" {
		addresses = append(addresses, msg.Transfer.Sender.Owner)
	}
	if msg.Transfer.Recipient.Owner != "" {
		addresses = append(addresses, msg.Transfer.Recipient.Owner)
	}
	return addresses
}

// extractPerpxSubaccountMsg handles subaccount deposit/withdraw messages
func extractPerpxSubaccountMsg(msgBytes json.RawMessage) []string {
	var msg struct {
		Sender struct {
			Owner string `json:"owner"`
		} `json:"sender"`
		Recipient struct {
			Owner string `json:"owner"`
		} `json:"recipient"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.Sender.Owner != "" {
		addresses = append(addresses, msg.Sender.Owner)
	}
	if msg.Recipient.Owner != "" {
		addresses = append(addresses, msg.Recipient.Owner)
	}
	return addresses
}

// extractPerpxOrderMsg handles order placement/cancellation
func extractPerpxOrderMsg(msgBytes json.RawMessage) []string {
	var msg struct {
		Order struct {
			OrderId struct {
				SubaccountId struct {
					Owner string `json:"owner"`
				} `json:"subaccount_id"`
			} `json:"order_id"`
		} `json:"order"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	if msg.Order.OrderId.SubaccountId.Owner != "" {
		return []string{msg.Order.OrderId.SubaccountId.Owner}
	}
	return nil
}

// extractPerpxProposedOperations handles proposed operations (usually empty or validator-related)
func extractPerpxProposedOperations(msgBytes json.RawMessage) []string {
	// This message type typically doesn't contain user addresses
	// It's used by validators for order matching
	return nil
}

// =============================================================================
// Standard Cosmos SDK extractors
// =============================================================================

// extractBankSend handles /cosmos.bank.v1beta1.MsgSend
func extractBankSend(msgBytes json.RawMessage) []string {
	var msg struct {
		FromAddress string `json:"from_address"`
		ToAddress   string `json:"to_address"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.FromAddress != "" {
		addresses = append(addresses, msg.FromAddress)
	}
	if msg.ToAddress != "" {
		addresses = append(addresses, msg.ToAddress)
	}
	return addresses
}

// extractBankMultiSend handles /cosmos.bank.v1beta1.MsgMultiSend
func extractBankMultiSend(msgBytes json.RawMessage) []string {
	var msg struct {
		Inputs []struct {
			Address string `json:"address"`
		} `json:"inputs"`
		Outputs []struct {
			Address string `json:"address"`
		} `json:"outputs"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	for _, input := range msg.Inputs {
		if input.Address != "" {
			addresses = append(addresses, input.Address)
		}
	}
	for _, output := range msg.Outputs {
		if output.Address != "" {
			addresses = append(addresses, output.Address)
		}
	}
	return addresses
}

// extractStakingDelegate handles delegate/undelegate messages
func extractStakingDelegate(msgBytes json.RawMessage) []string {
	var msg struct {
		DelegatorAddress string `json:"delegator_address"`
		ValidatorAddress string `json:"validator_address"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.DelegatorAddress != "" {
		addresses = append(addresses, msg.DelegatorAddress)
	}
	// Note: validator addresses have a different prefix, but we include them anyway
	if msg.ValidatorAddress != "" {
		addresses = append(addresses, msg.ValidatorAddress)
	}
	return addresses
}

// extractStakingRedelegate handles redelegate messages
func extractStakingRedelegate(msgBytes json.RawMessage) []string {
	var msg struct {
		DelegatorAddress    string `json:"delegator_address"`
		ValidatorSrcAddress string `json:"validator_src_address"`
		ValidatorDstAddress string `json:"validator_dst_address"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.DelegatorAddress != "" {
		addresses = append(addresses, msg.DelegatorAddress)
	}
	if msg.ValidatorSrcAddress != "" {
		addresses = append(addresses, msg.ValidatorSrcAddress)
	}
	if msg.ValidatorDstAddress != "" {
		addresses = append(addresses, msg.ValidatorDstAddress)
	}
	return addresses
}

// extractDistributionWithdraw handles distribution withdraw messages
func extractDistributionWithdraw(msgBytes json.RawMessage) []string {
	var msg struct {
		DelegatorAddress string `json:"delegator_address"`
		ValidatorAddress string `json:"validator_address"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.DelegatorAddress != "" {
		addresses = append(addresses, msg.DelegatorAddress)
	}
	if msg.ValidatorAddress != "" {
		addresses = append(addresses, msg.ValidatorAddress)
	}
	return addresses
}

// extractGovVote handles governance vote messages
func extractGovVote(msgBytes json.RawMessage) []string {
	var msg struct {
		Voter string `json:"voter"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	if msg.Voter != "" {
		return []string{msg.Voter}
	}
	return nil
}

// extractGovDeposit handles governance deposit messages
func extractGovDeposit(msgBytes json.RawMessage) []string {
	var msg struct {
		Depositor string `json:"depositor"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	if msg.Depositor != "" {
		return []string{msg.Depositor}
	}
	return nil
}

// extractGovProposal handles governance proposal messages
func extractGovProposal(msgBytes json.RawMessage) []string {
	var msg struct {
		Proposer string `json:"proposer"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	if msg.Proposer != "" {
		return []string{msg.Proposer}
	}
	return nil
}

// extractAuthzGrant handles authz grant/revoke messages
func extractAuthzGrant(msgBytes json.RawMessage) []string {
	var msg struct {
		Granter string `json:"granter"`
		Grantee string `json:"grantee"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.Granter != "" {
		addresses = append(addresses, msg.Granter)
	}
	if msg.Grantee != "" {
		addresses = append(addresses, msg.Grantee)
	}
	return addresses
}

// extractAuthzExec handles authz exec messages (extracts grantee and inner message addresses)
func extractAuthzExec(msgBytes json.RawMessage) []string {
	var msg struct {
		Grantee string            `json:"grantee"`
		Msgs    []json.RawMessage `json:"msgs"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.Grantee != "" {
		addresses = append(addresses, msg.Grantee)
	}

	// Extract addresses from inner messages using generic extractor
	for _, innerMsg := range msg.Msgs {
		innerAddresses := genericExtractor(innerMsg)
		addresses = append(addresses, innerAddresses...)
	}

	return addresses
}

// extractFeeGrant handles fee grant messages
func extractFeeGrant(msgBytes json.RawMessage) []string {
	var msg struct {
		Granter string `json:"granter"`
		Grantee string `json:"grantee"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.Granter != "" {
		addresses = append(addresses, msg.Granter)
	}
	if msg.Grantee != "" {
		addresses = append(addresses, msg.Grantee)
	}
	return addresses
}

// extractIBCTransfer handles IBC transfer messages
func extractIBCTransfer(msgBytes json.RawMessage) []string {
	var msg struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
	}

	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil
	}

	var addresses []string
	if msg.Sender != "" {
		addresses = append(addresses, msg.Sender)
	}
	if msg.Receiver != "" {
		addresses = append(addresses, msg.Receiver)
	}
	return addresses
}

// =============================================================================
// Generic fallback extractor
// =============================================================================

// addressFieldNames defines the JSON fields that typically contain account addresses.
var addressFieldNames = map[string]bool{
	"signer": true, "sender": true, "receiver": true, "to_address": true, "from_address": true,
	"delegator_address": true, "validator_address": true, "submitter": true, "proposer": true,
	"depositor": true, "voter": true, "validator_dst_address": true, "validator_src_address": true,
	"grantee": true, "granter": true, "payer": true, "admin": true, "authority": true, "creator": true,
	"owner": true, "recipient": true, "spender": true, "user": true, "address": true,
}

// genericExtractor is the fallback that recursively searches for address fields.
func genericExtractor(msgBytes json.RawMessage) []string {
	var msgMap map[string]interface{}
	if err := json.Unmarshal(msgBytes, &msgMap); err != nil {
		return nil
	}

	var addresses []string
	extractAddressesRecursiveGeneric(msgMap, &addresses)
	return addresses
}

// extractAddressesRecursiveGeneric recursively searches a map for address fields.
func extractAddressesRecursiveGeneric(data map[string]interface{}, addresses *[]string) {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			if addressFieldNames[key] && isValidAddressGeneric(v) {
				*addresses = append(*addresses, v)
			}
		case map[string]interface{}:
			extractAddressesRecursiveGeneric(v, addresses)
		case []interface{}:
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					extractAddressesRecursiveGeneric(itemMap, addresses)
				}
			}
		}
	}
}

// isValidAddressGeneric checks if a string looks like a valid bech32 address.
func isValidAddressGeneric(s string) bool {
	// Basic validation: addresses are typically 39-65 characters
	if len(s) < 39 || len(s) > 65 {
		return false
	}

	// Check for bech32 format: prefix + "1" + data
	idx := strings.Index(s, "1")
	if idx > 0 && idx < 20 {
		return true
	}

	return false
}
