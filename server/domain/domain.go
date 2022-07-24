// domain содержит основные типы с которыми работает сервер.
package domain

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

// Rate представляет собой обменный курс
// одной конкретной валюты (по отношению к рублю).
type Rate struct {
	Id       int64   `json:"id,omitempty"`
	CharCode string  `json:"char_code,omitempty"`
	Nominal  int     `json:"nominal,omitempty"`
	Time     int64   `json:"time"`
	Value    float64 `json:"value"`
}

// JsonRate - структура для парсинга курса BTC/USDT;
// нас интересует только курс и время.
type JsonRate struct {
	Time int64   `json:"time"`
	Buy  float64 `json:"buy,string"`
}

// XMLRate - структура для парсинга курса фиатных валют.
type XMLRate struct {
	XMLName  xml.Name   `xml:"Valute"`
	CharCode string     `xml:"CharCode"`
	Nominal  int        `xml:"Nominal"`
	Time     int64      `xml:"-"`
	Value    commaFloat `xml:"Value"`
}

// ToRate - приводит JsonRate к Rate.
func (br *JsonRate) ToRate() Rate {
	return Rate{
		Time:  time.UnixMilli(br.Time).Unix(),
		Value: br.Buy,
	}
}

// ToRate - приводит XMLRate к Rate.
func (r *XMLRate) ToRate() Rate {
	return Rate{
		Id:       0,
		CharCode: r.CharCode,
		Nominal:  r.Nominal,
		Time:     r.Time,
		Value:    float64(r.Value),
	}
}

type commaFloat float64

type xmlContainer struct {
	Items []Rate `xml:"Valute"`
}

// RateMap превращает слайс Rate в map.
func RateMap(rates []Rate) map[string]any {
	m := make(map[string]any, len(rates))
	for i := range rates {
		m[rates[i].CharCode] = rates[i].Value
	}
	return m
}

// RateMap превращает слайс Rate в map с таймстампом.
func RateMapTimestamp(rates []Rate) map[string]any {
	m := make(map[string]any, len(rates))
	for i := range rates {
		m["timestamp"] = rates[i].Time
		m["value"] = rates[i].Value
	}
	return m
}

func (r *Rate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var xr XMLRate
	err := d.DecodeElement(&xr, &start)
	if err != nil {
		return err
	}
	*r = xr.ToRate()
	t := time.Now()
	r.Time = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
	return nil
}

func (c *commaFloat) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	err := d.DecodeElement(&s, &start)
	if err != nil {
		return err
	}
	f, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	if err != nil {
		return err
	}
	*c = commaFloat(f)
	return nil
}

// JsonDec - логика десериализации json данных
func JsonDec(b []byte) ([]Rate, error) {
	var c = struct {
		Item JsonRate `json:"data"`
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
	return c.Items, xmlDecoderWithSettings(bytes.NewReader(b)).Decode(&c)
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

// decoderWithSettings возвращает *xml.Decoder с настройками
func xmlDecoderWithSettings(r io.Reader) *xml.Decoder {
	decoder := xml.NewDecoder(r)
	decoder.CharsetReader = charset.NewReaderLabel // некоторые возвращают не UTF-8
	return decoder
}
