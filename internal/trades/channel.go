package trades

import (
	"regexp"
	"strings"
)

const (
	ChannelLiga   = "liga"
	ChannelMyP    = "myp"
	ChannelSite   = "site"
	ChannelBalcao = "balcao"
	ChannelOutros = "outros"
)

const (
	WriteOffAberto = "aberto"
	WriteOffPerda  = "perda"
	WriteOffBrinde = "brinde"
	WriteOffUso    = "uso"
)

var (
	ligaOrder  = regexp.MustCompile(`(?i)\bliga\b`)
	mypOrder   = regexp.MustCompile(`(?i)\bmyp #\d+`)
	siteOrder  = regexp.MustCompile(`(?i)(\bsite #\d+|^site$)`)
	directSale = regexp.MustCompile(`(?i)(balc[aã]o|^myp( cards)?$|por fora|abert[oa])`)
	openedCase = regexp.MustCompile(`(?i)abert[oa]`)
)

func ValidChannel(c string) bool {
	switch c {
	case "", ChannelLiga, ChannelMyP, ChannelSite, ChannelBalcao, ChannelOutros:
		return true
	}
	return false
}

func ValidWriteOff(w string) bool {
	switch w {
	case "", WriteOffAberto, WriteOffPerda, WriteOffBrinde, WriteOffUso:
		return true
	}
	return false
}

func ChannelOf(t Trade) string {
	if t.Channel != "" {
		return t.Channel
	}
	b := strings.TrimSpace(t.Buyer)
	switch {
	case mypOrder.MatchString(b):
		return ChannelMyP
	case siteOrder.MatchString(b):
		return ChannelSite
	case ligaOrder.MatchString(b):
		return ChannelLiga
	case directSale.MatchString(b):
		return ChannelBalcao
	}
	return ChannelOutros
}

func WriteOffOf(t Trade) string {
	if t.WriteOff != "" {
		return t.WriteOff
	}
	if t.Channel == "" && openedCase.MatchString(t.Buyer) {
		return WriteOffAberto
	}
	return ""
}
