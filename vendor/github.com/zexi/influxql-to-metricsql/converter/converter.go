package converter

import (
	"io"
	"strings"

	"github.com/influxdata/influxql"
	"github.com/pkg/errors"

	"github.com/zexi/influxql-to-metricsql/converter/translator"
)

type Converter interface {
	Translate() (string, error)
	TranslateWithTimeRange() (string, *influxql.TimeRange, error)
}

type converter struct {
	influxParser *influxql.Parser
	translator   translator.Translator
}

func Translate(influxQL string) (string, error) {
	return New(strings.NewReader(influxQL)).Translate()
}

func TranslateWithTimeRange(influxQL string) (string, *influxql.TimeRange, error) {
	return New(strings.NewReader(influxQL)).TranslateWithTimeRange()
}

func New(r io.Reader) Converter {
	c := &converter{
		influxParser: influxql.NewParser(r),
		translator:   translator.NewPromQL(),
	}
	return c
}

func (c converter) Translate() (string, error) {
	q, err := c.influxParser.ParseQuery()
	if err != nil {
		return "", errors.Wrap(err, "influxParser.ParserQuery")
	}
	if len(q.Statements) > 1 {
		return "", errors.Errorf("Only support 1 statement translating")
	}
	return c.translator.Translate(q.Statements[0])
}

func (c converter) TranslateWithTimeRange() (string, *influxql.TimeRange, error) {
	promQL, err := c.Translate()
	if err != nil {
		return "", nil, errors.Wrap(err, "Translate")
	}
	return promQL, c.translator.GetTimeRange(), nil
}
