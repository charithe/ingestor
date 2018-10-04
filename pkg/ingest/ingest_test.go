package ingest

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/charithe/ingestor/pkg/update"
	"github.com/stretchr/testify/assert"
)

func TestFormatMobileNumber(t *testing.T) {
	testCases := []struct {
		Input string
		Want  string
	}{
		{
			Input: "(013890) 37420",
			Want:  "01389037420",
		},
		{
			Input: "0800 1234 5679",
			Want:  "080012345679",
		},
		{
			Input: "442345 354566",
			Want:  "2345354566",
		},
		{
			Input: "0044 56789 3456",
			Want:  "567893456",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			have := formatMobileNumber(tc.Input)
			assert.Equal(t, tc.Want, have)
		})
	}
}

type mockUpdater struct {
	records []*update.Record
}

func (m *mockUpdater) Update(ctx context.Context, rec *update.Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if rec == nil || rec.Id < 0 {
		return fmt.Errorf("error updating record")
	}

	m.records = append(m.records, rec)
	return nil
}

func TestIngestor(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		f, err := os.Open("testdata/input1.csv")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		m := &mockUpdater{}
		ing := New(f, m)
		if err := ing.Start(context.Background()); err != nil {
			t.Fatalf("Ingest failed: %+v", err)
		}

		assert.Len(t, m.records, 3)

		assert.EqualValues(t, 1, m.records[0].Id)
		assert.Equal(t, "Kirk", m.records[0].Name)
		assert.Equal(t, "ornare@sedtortor.net", m.records[0].Email)
		assert.Equal(t, "01389037420", m.records[0].MobileNumber)

		assert.EqualValues(t, 2, m.records[1].Id)
		assert.Equal(t, "Cain", m.records[1].Name)
		assert.Equal(t, "volutpat@semmollisdui.com", m.records[1].Email)
		assert.Equal(t, "0169772245", m.records[1].MobileNumber)

		assert.EqualValues(t, 3, m.records[2].Id)
		assert.Equal(t, "Geoffrey", m.records[2].Name)
		assert.Equal(t, "vitae@consectetuermaurisid.co.uk", m.records[2].Email)
		assert.Equal(t, "08001111", m.records[2].MobileNumber)
	})

	t.Run("halfway_cancellation", func(t *testing.T) {
		// TODO implement context cancellation test
	})

}
