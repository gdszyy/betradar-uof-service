package services

// OddsCompetitor 赔率消息中的参赛队伍
type OddsCompetitor struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	Qualifier string `xml:"qualifier,attr"` // home/away
}

