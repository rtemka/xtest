package domain

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"sync"
	"testing"
	"time"
)

const blob = `
		<ValCurs>
			<Valute>
				<NumCode>036</NumCode>
				<CharCode>AUD</CharCode>
				<Nominal>1</Nominal>
				<Name>Австралийский доллар</Name>
				<Value>37,9799</Value>
			</Valute>
			<Valute>
				<NumCode>036</NumCode>
				<CharCode>AUD</CharCode>
				<Nominal>1</Nominal>
				<Name>Австралийский доллар</Name>
				<Value>37,9799</Value>
			</Valute>
		</ValCurs>
`

const jblob = `
		{
			"code": "200000",
			"data": {
			"time": 1658237004004,
			"symbol": "BTC-USDT",
			"buy": "22278.80"
			}
		}`

func TestRate_UnmarshalXML(t *testing.T) {

	r := struct {
		Items []Rate `xml:">Valute"`
	}{}

	err := xml.NewDecoder(strings.NewReader(blob)).Decode(&r)
	if err != nil {
		t.Fatalf("UnmarshalXML() error = %v", err)
	}

	tn := time.Now()
	want := Rate{
		Id:       0,
		CharCode: "AUD",
		Nominal:  1,
		Time:     time.Date(tn.Year(), tn.Month(), tn.Day(), 0, 0, 0, 0, time.UTC).Unix(),
		Value:    37.9799,
	}

	for _, got := range r.Items {
		if got != want {
			t.Fatalf("UnmarshalXML() got = %#v, want %#v", got, want)
		}
	}

}

func TestRate_UnmarshalJSON(t *testing.T) {

	got := struct {
		Item BtcRate `json:"data"`
	}{}

	err := json.NewDecoder(strings.NewReader(jblob)).Decode(&got)
	if err != nil {
		t.Log(got)
		t.Fatalf("UnmarshalJSON() = error %v", err)
	}

	want := Rate{
		Id:       0,
		CharCode: "",
		Nominal:  0,
		Time:     1658237004004,
		Value:    22278.80,
	}

	if got.Item.ToRate() != want {
		t.Fatalf("UnmarshalJSON() = %#v, want %#v", got.Item, want)
	}

}

func TestDecodeStream(t *testing.T) {

	t.Run("xmlDecode", func(t *testing.T) {
		tn := time.Now()
		var want = Rate{
			Id:       0,
			CharCode: "AUD",
			Nominal:  1,
			Time:     time.Date(tn.Year(), tn.Month(), tn.Day(), 0, 0, 0, 0, time.UTC).Unix(),
			Value:    37.9799,
		}

		ch := make(chan []byte)
		vals, errs := DecodeStream(ch, XmlDec)

		go func() {
			for i := 0; i < 10; i++ {
				ch <- []byte(blob)
			}
			close(ch)
		}()

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			for err := range errs {
				t.Errorf("DecodeStream() = error %v", err)
			}
			wg.Done()
		}()

		for got := range vals {
			for _, got := range got {
				if got != want {
					t.Errorf("DecodeStream() got = %#v, want %#v", got, want)
				}
			}
		}

		wg.Wait()
	})

	t.Run("jsonDecode", func(t *testing.T) {
		want := Rate{
			Id:       0,
			CharCode: "",
			Nominal:  0,
			Time:     1658237004004,
			Value:    22278.80,
		}

		ch := make(chan []byte)
		vals, errs := DecodeStream(ch, JsonDec)

		go func() {
			for i := 0; i < 10; i++ {
				ch <- []byte(jblob)
			}
			close(ch)
		}()

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			for err := range errs {
				t.Errorf("DecodeStream() = error %v", err)
			}
			wg.Done()
		}()

		for got := range vals {
			for _, got := range got {
				if got != want {
					t.Errorf("DecodeStream() got = %#v, want %#v", got, want)
				}
			}
		}

		wg.Wait()
	})

}
