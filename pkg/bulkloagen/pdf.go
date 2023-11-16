package bulkloagen

import (
	"bytes"
	"text/template"
	"time"

	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
)

const (
	templMain    = "Please use this letter as authorization to {{.Partner}} (Partner) or their designated agent(s) to order/run a cross-connect towards the following demarcation point of {{.OurName}}. This LOA does not obligate {{.OurName}} to be billed for any services."
	templExpiry  = "This LOA expires {{.ExpiryDays}} calendar days from the date of issue if not used, upon notification to Partner by {{.OurName}} or on the date that the cross-connect is installed, whichever is earlier. It is not automatically re-usable by Partner or their designated agent(s)."
	templContact = "Please contact the {{.OurNameShort}} NOC after the work has been done with information about the used port. Should you have any questions or concern regarding this LOA please contact {{.OurNameShort}} NOC."
)

type TemplateData struct {
	Site          string
	DemarcPanel   string
	DemarcPort    string
	Device        string
	RearPorts     map[int64]string
	RearPortID    int
	OurName       string
	OurNameShort  string
	OurStreet     string
	OurCity       string
	OurNocName    string
	OurNocEmail   string
	OurNocPhone   string
	Partner       string
	PartnerStreet string
	PartnerCity   string
	ExpiryDays    int

	ExampleDeviceURL string
	Version          string
}

func Generate(data TemplateData) (*bytes.Buffer, error) {
	t := template.Must(template.New("main").Parse(templMain))
	template.Must(t.New("expiry").Parse(templExpiry))
	template.Must(t.New("contact").Parse(templContact))

	outMain := bytes.NewBuffer(make([]byte, 0))
	if err := t.ExecuteTemplate(outMain, "main", data); err != nil {
		return nil, err
	}

	outExpiry := bytes.NewBuffer(make([]byte, 0))
	if err := t.ExecuteTemplate(outExpiry, "expiry", data); err != nil {
		return nil, err
	}

	outContact := bytes.NewBuffer(make([]byte, 0))
	if err := t.ExecuteTemplate(outContact, "contact", data); err != nil {
		return nil, err
	}

	m := pdf.NewMaroto(consts.Portrait, consts.A4)
	m.SetPageMargins(20, 20, 20)

	m.RegisterHeader(func() {
		m.Row(20, func() {
			m.Col(8, func() {
				m.Text(data.Partner, props.Text{
					Size:  10,
					Style: consts.BoldItalic,
					Align: consts.Left,
				})
				m.Text(data.PartnerStreet, props.Text{
					Top:   4,
					Size:  10,
					Align: consts.Left,
				})
				m.Text(data.PartnerCity, props.Text{
					Top:   8,
					Size:  10,
					Align: consts.Left,
				})
			})

			m.ColSpace(1)

			m.Col(3, func() {
				m.Text(data.OurName, props.Text{
					Size:  10,
					Style: consts.BoldItalic,
					Align: consts.Right,
				})
				m.Text(data.OurStreet, props.Text{
					Top:   4,
					Size:  10,
					Align: consts.Right,
				})
				m.Text(data.OurCity, props.Text{
					Top:   8,
					Size:  10,
					Align: consts.Right,
				})
			})
		})
	})

	m.RegisterFooter(func() {})

	m.Row(15, func() {
		m.Col(6, func() {
			m.Text("Letter of Authorization", props.Text{Style: consts.Bold})
		})
		m.Col(6, func() {
			m.Text(time.Now().Format("02. January 2006"), props.Text{Align: consts.Right})
		})
	})

	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("To whom it may concern:", props.Text{})
		})
	})

	m.Row(15, func() {
		m.Col(12, func() {
			m.Text(outMain.String(), props.Text{})
		})
	})

	m.Row(10, func() {
		m.TableList([]string{"", "", ""}, [][]string{
			{"", "Site", data.Site},
			{"", "Demarcation Panel", data.DemarcPanel},
			{"", "Demarcation Port", data.DemarcPort},
		}, props.TableList{
			ContentProp: props.TableListContent{
				Size:      10,
				GridSizes: []uint{1, 3, 8},
			},
			Align:              consts.Left,
			HeaderContentSpace: 1,
			Line:               false,
		})
	})

	m.Row(2, func() {})
	m.Row(15, func() {
		m.Col(12, func() {
			m.Text(outExpiry.String(), props.Text{})
		})
	})

	m.Row(15, func() {
		m.Col(12, func() {
			m.Text(outContact.String(), props.Text{})
		})
	})

	m.Row(20, func() {
		m.ColSpace(1)
		m.Col(11, func() {
			m.Text(data.OurNocName, props.Text{})
			m.Text(data.OurNocEmail, props.Text{Top: 5})
			m.Text(data.OurNocPhone, props.Text{Top: 10})
		})
	})

	m.Row(15, func() {
		m.Col(12, func() {
			m.Text("Yours sincerely,", props.Text{})
			m.Text(data.OurName, props.Text{Top: 8, Style: consts.BoldItalic})
		})
	})

	buf, err := m.Output()
	if err != nil {
		return nil, err
	}

	return &buf, nil
}
