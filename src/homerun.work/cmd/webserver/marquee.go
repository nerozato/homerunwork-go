package main

//Marquee : definition of a marquee
type Marquee struct {
	Text    string
	URL     string
	URLText string
}

//GenerateMarqueeProvider : generate a marquee based on the provider state
func GenerateMarqueeProvider(provider *providerUI, count int) *Marquee {
	//check for new bookings
	if count > 0 {
		var text string
		if count == 1 {
			text = GetMsgText(MsgBookingNewSingle)
		} else {
			text = GetMsgText(MsgBookingNewMultiple, count)
		}
		marquee := &Marquee{
			Text:    text,
			URL:     provider.GetURLBookings(),
			URLText: "View",
		}
		return marquee
	}
	return nil
}
