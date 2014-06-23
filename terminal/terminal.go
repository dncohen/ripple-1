// Utiltities for formatting Ripple data in a terminal
package terminal

import (
	"fmt"
	"github.com/donovanhide/ripple/data"
	"github.com/donovanhide/ripple/websockets"
	"github.com/fatih/color"
	"reflect"
)

type Flag uint32

const (
	Indent Flag = 1 << iota
	DoubleIndent
	TripleIndent

	ShowLedgerSequence
)

var Default Flag

var (
	ledgerStyle     = color.New(color.FgRed, color.Underline)
	leStyle         = color.New(color.FgWhite)
	txStyle         = color.New(color.FgGreen)
	proposalStyle   = color.New(color.FgYellow)
	validationStyle = color.New(color.FgYellow, color.Bold)
	tradeStyle      = color.New(color.FgBlue)
	balanceStyle    = color.New(color.FgMagenta)
	pathStyle       = color.New(color.FgYellow)
	infoStyle       = color.New(color.FgRed)
)

type bundle struct {
	color  *color.Color
	format string
	values []interface{}
	flag   Flag
}

func newLeBundle(v interface{}, flag Flag) (*bundle, error) {
	var (
		format = "%-11s "
		values = []interface{}{v.(data.LedgerEntry).GetLedgerEntryType()}
	)
	switch le := v.(type) {
	case *data.AccountRoot:
		format += "%-34s %08X %s"
		values = append(values, []interface{}{le.Account, *le.Flags, le.Balance}...)
	case *data.LedgerHashes:
		format += "%d hashes"
		values = append(values, []interface{}{len(le.Hashes)}...)
	case *data.RippleState:
		format += "%s %s %s"
		values = append(values, []interface{}{le.Balance, le.HighLimit, le.LowLimit}...)
	case *data.Offer:
		format += "%-34s %-60s %-60s %-18s"
		values = append(values, []interface{}{le.Account, le.TakerPays, le.TakerGets, le.Ratio()}...)
	case *data.FeeSettings:
		format += "%d %d %d %d"
		values = append(values, []interface{}{le.BaseFee, le.ReferenceFeeUnits, le.ReserveBase, le.ReserveIncrement}...)
	case *data.Amendments:
		format += "%s"
		values = append(values, []interface{}{le.Amendments}...)
	default:
		return nil, fmt.Errorf("Unknown Ledger Entry Type")
	}
	return &bundle{
		color:  leStyle,
		format: format,
		values: values,
		flag:   flag,
	}, nil
}

func newTxBundle(v data.Transaction, insert string, flag Flag) (*bundle, error) {
	var (
		base   = v.GetBase()
		format = "%-11s %-8s %s%s %-34s "
		values = []interface{}{base.GetType(), base.Fee, insert, base.MemoSymbol(), base.Account}
	)
	switch tx := v.(type) {
	case *data.Payment:
		format += "=> %-34s %-60s %-60s"
		values = append(values, []interface{}{tx.Destination, tx.Amount, tx.SendMax}...)
	case *data.OfferCreate:
		format += "%-60s %-60s %-18s"
		values = append(values, []interface{}{tx.TakerPays, tx.TakerGets, tx.Ratio()}...)
	case *data.OfferCancel:
		format += "%-9d"
		values = append(values, tx.Sequence)
	case *data.AccountSet:
		format += "%-9d"
		values = append(values, tx.Sequence)
	case *data.TrustSet:
		format += "%-60s %d %d"
		values = append(values, tx.LimitAmount, tx.QualityIn, tx.QualityOut)
	}
	return &bundle{
		color:  txStyle,
		format: format,
		values: values,
		flag:   flag,
	}, nil
}

func newTxmBundle(txm *data.TransactionWithMetaData, flag Flag) (*bundle, error) {
	insert := fmt.Sprintf("%s ", txm.MetaData.TransactionResult.Symbol())
	if flag&ShowLedgerSequence > 0 {
		insert = fmt.Sprintf("%-9d %s", txm.LedgerSequence, insert)
	}
	b, err := newTxBundle(txm.Transaction, insert, flag)
	if err != nil {
		return nil, err
	}
	if !txm.MetaData.TransactionResult.Success() {
		b.color = infoStyle
	}
	return b, nil
}

func newBundle(value interface{}, flag Flag) (*bundle, error) {
	switch v := value.(type) {
	case *data.TransactionWithMetaData:
		return newTxmBundle(v, flag)
	case data.Transaction:
		return newTxBundle(v, "", flag)
	case data.LedgerEntry:
		return newLeBundle(v, flag)
	}
	switch v := reflect.Indirect(reflect.ValueOf(value)).Interface().(type) {
	case websockets.LedgerStreamMsg:
		return &bundle{
			color:  ledgerStyle,
			format: "Ledger %d closed at %s with %d transactions",
			values: []interface{}{v.LedgerSequence, v.LedgerTime.String(), v.TxnCount},
			flag:   flag,
		}, nil
	case websockets.ServerStreamMsg:
		return &bundle{
			color:  infoStyle,
			format: "Server Status: %s (%d/%d)",
			values: []interface{}{v.Status, v.LoadFactor, v.LoadBase},
			flag:   flag,
		}, nil
	case data.Ledger:
		return &bundle{
			color:  ledgerStyle,
			format: "Ledger %d closed at %s",
			values: []interface{}{v.LedgerSequence, v.CloseTime.String()},
			flag:   flag,
		}, nil
	case data.InnerNode:
		return &bundle{
			color:  leStyle,
			format: "%s: %d hashes",
			values: []interface{}{v.Type, v.Count()},
			flag:   flag,
		}, nil
	case data.Proposal:
		return &bundle{
			color:  proposalStyle,
			format: "%s Proposal:   %s %s %s %s",
			values: []interface{}{data.CheckSymbol(&v), v.PublicKey.NodePublicKey(), v.CloseTime.String(), v.PreviousLedger, v.LedgerHash},
			flag:   flag,
		}, nil
	case data.Validation:
		return &bundle{
			color:  validationStyle,
			format: "%s Validation: %s %s %s %-8d %08X %s",
			values: []interface{}{data.CheckSymbol(&v), v.SigningPubKey.NodePublicKey(), v.SigningTime.String(), v.LedgerHash, v.LedgerSequence, v.Flags, v.Amendments},
			flag:   flag,
		}, nil
	case data.Trade:
		return &bundle{
			color:  tradeStyle,
			format: "Trade: %-34s => %-34s %-18s %60s => %-60s",
			values: []interface{}{v.Seller, v.Buyer, v.Price(), v.Paid, v.Got},
			flag:   flag,
		}, nil
	case data.Balance:
		return &bundle{
			color:  balanceStyle,
			format: "Balance: %-34s  Currency: %s Balance: %20s Change: %20s",
			values: []interface{}{v.Account, v.Currency, v.Balance, v.Change},
			flag:   flag,
		}, nil
	case data.Paths:
		sig, err := v.Signature()
		if err != nil {
			return nil, err
		}
		return &bundle{
			color:  pathStyle,
			format: "Path: %08X %s",
			values: []interface{}{sig, v.String()},
			flag:   flag,
		}, nil
	default:
		return &bundle{
			color:  infoStyle,
			format: "%s",
			values: []interface{}{v},
			flag:   flag,
		}, nil
	}
}

func indent(flag Flag) string {
	switch {
	case flag&Indent > 0:
		return "  "
	case flag&DoubleIndent > 0:
		return "    "
	case flag&TripleIndent > 0:
		return "      "
	default:
		return ""
	}
}

func println(value interface{}, flag Flag) (int, error) {
	b, err := newBundle(value, flag)
	if err != nil {
		return 0, err
	}
	return b.color.Printf(indent(flag)+b.format+"\n", b.values...)
}

func Println(value interface{}, flag Flag) {
	if _, err := println(value, flag); err != nil {
		infoStyle.Println(err.Error())
	}
}

func Sprint(value interface{}, flag Flag) string {
	b, err := newBundle(value, flag)
	if err != nil {
		return fmt.Sprintf("Cannot format: %+v", value)
	}
	return b.color.SprintfFunc()(indent(flag)+b.format, b.values...)
}