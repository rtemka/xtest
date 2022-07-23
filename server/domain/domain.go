// domain содержит основные типы с которыми работает сервер.
package domain

import (
	"encoding/json"
	"encoding/xml"
	"time"
)

// Rate представляет собой обменный курс
// одной конкретной валюты (по отношению к рублю).
type Rate struct {
	Id       int64   `json:"id" xml:"-"`
	CharCode string  `json:"char_code,omitempty" xml:"CharCode"`
	Nominal  int     `json:"nominal,omitempty" xml:"Nominal"`
	Time     int64   `json:"time" xml:"-"`
	Value    float64 `json:"value" xml:"Value"`
}

// BtcRate - структура для парсинга курса BTC/USDT;
// нас интересует только курс и время.
type BtcRate struct {
	Time int64   `json:"time"`
	Buy  float64 `json:"buy,string"`
}

type xmlContainer struct {
	Items []Rate `xml:">Valute"`
}

// ToRate - приводит BtcRate к Rate.
func (br *BtcRate) ToRate() Rate {
	return Rate{
		Time:  br.Time,
		Value: br.Buy,
	}
}

func (r *Rate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	err := d.DecodeElement(r, &start)
	if err != nil {
		return err
	}
	t := time.Now()
	r.Time = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
	return nil
}

// JsonDec - логика десериализации json данных
func JsonDec(b []byte) ([]Rate, error) {
	var c = struct {
		Item BtcRate `json:"data"`
	}{}
	err := json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	return []Rate{c.Item.ToRate()}, nil
}

// XmlDec - логика десериализации xml данных
func XmlDec(b []byte) ([]Rate, error) {
	var c xmlContainer
	return c.Items, xml.Unmarshal(b, &c)
}

// DecodeStream - читает из канала поток срезов байтов,
// десериализует их с помощью предоставленной функции
// (JsonDec/XmlDec) и отправляет дальше по каналу.
func DecodeStream(in <-chan []byte, f func([]byte) ([]Rate, error)) (<-chan []Rate, <-chan error) {

	out := make(chan []Rate)
	errs := make(chan error, 2)

	go func() {

		defer func() {
			close(errs)
			close(out)
		}()

		for v := range in {
			r, err := f(v)
			if err == nil {
				out <- r
			} else {
				errs <- err
			}
		}
	}()

	return out, errs
}

// func (r *Rates) marshalJSONCurrLatest() ([]byte, error) {
// 	m := make(map[string]any, 1)
// 	m[r.CharCode] = r.Value
// 	return json.Marshal(m)
// }

// func (r *Rate) MarshalJSON() ([]byte, error) {
// 	switch r.Mode {
// 	case BtcHistory:
// 		return r.marshalJSONBtcHistory()
// 	case CurrLatest:
// 		return r.marshalJSONCurrLatest()
// 	default:
// 		return json.Marshal(r)
// 	}
// }
