package ipace

import (
	"os"
	"testing"
	"time"
)

func sampleInt(value int) *int           { return &value }
func sampleFloat(value float64) *float64 { return &value }
func sampleBool(value bool) *bool        { return &value }

func TestGeneratePublicSampleWorkbook(t *testing.T) {
	if os.Getenv("GENERATE_MEMBER_EXPORT_SAMPLE") != "1" {
		t.Skip("set GENERATE_MEMBER_EXPORT_SAMPLE=1 to regenerate the public sample")
	}
	generated := time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)
	joined := time.Date(2026, 7, 18, 14, 15, 0, 0, time.UTC)
	snapshot := memberSnapshot{
		Email:       "alex.owner@example.test",
		GeneratedAt: generated,
		JoinRecords: []joinRecord{{
			CreatedAt: joined,
			UpdatedAt: joined,
			Contact: contactRecord{
				Name:    "Alex Owner",
				Country: "United Kingdom",
			},
			Membership: membershipRecord{
				Relationship: "Current owner",
				Skills:       []string{"Community support", "Technical"},
			},
			Consents: consentRecord{
				Contact:            true,
				AnonymisedAnalysis: true,
				NotLegalClaim:      true,
			},
		}},
		VehicleRecords: []vehicleRecord{
			{
				ID:        "sample-vehicle-one",
				CreatedAt: joined,
				UpdatedAt: generated,
				Vehicle: vehicleDetails{
					VINLast6:              "123456",
					Registration:          "SAMPLE1",
					Country:               "United Kingdom",
					ModelYear:             "2020",
					Mileage:               sampleInt(48520),
					OwnedSince:            "2021-03-12",
					FirstRegistrationDate: "2020-11-06",
				},
				Battery: batteryDetails{
					StateOfHealth:        sampleFloat(88.4),
					MeasuredAt:           "2026-07-20",
					MileageAtMeasurement: sampleInt(48380),
					Source:               "Independent diagnostic",
				},
			},
			{
				ID:        "sample-vehicle-two",
				CreatedAt: joined.Add(24 * time.Hour),
				UpdatedAt: generated,
				Vehicle: vehicleDetails{
					VINLast6:              "654321",
					Registration:          "SAMPLE2",
					Country:               "United Kingdom",
					ModelYear:             "2022",
					Mileage:               sampleInt(21840),
					OwnedSince:            "2024-01-20",
					FirstRegistrationDate: "2022-05-17",
				},
				Battery: batteryDetails{
					StateOfHealth:        sampleFloat(94.1),
					MeasuredAt:           "2026-07-22",
					MileageAtMeasurement: sampleInt(21790),
					Source:               "Dealer report",
				},
			},
		},
	}
	for index, reading := range []struct {
		vehicleID string
		date      string
		soh       float64
		mileage   int
		source    string
	}{
		{"sample-vehicle-one", "2025-10-03", 92.7, 39120, "Dealer report"},
		{"sample-vehicle-one", "2026-01-16", 91.2, 42460, "Independent diagnostic"},
		{"sample-vehicle-one", "2026-07-20", 88.4, 48380, "Independent diagnostic"},
		{"sample-vehicle-two", "2025-08-12", 96.3, 13250, "Dealer report"},
		{"sample-vehicle-two", "2026-07-22", 94.1, 21790, "Dealer report"},
	} {
		snapshot.BatteryReadings = append(snapshot.BatteryReadings, batteryReadingRecord{
			ID:        "sample-reading-" + string(rune('a'+index)),
			VehicleID: reading.vehicleID,
			CreatedAt: generated,
			UpdatedAt: generated,
			Battery: batteryDetails{
				StateOfHealth:        sampleFloat(reading.soh),
				MeasuredAt:           reading.date,
				MileageAtMeasurement: sampleInt(reading.mileage),
				Source:               reading.source,
			},
		})
	}
	for index, event := range []serviceEventRecord{
		{VehicleID: "sample-vehicle-one", EventType: "fault", OccurredAt: "2025-11-14", Mileage: sampleInt(40230), Title: "Traction battery warning", Description: "Warning displayed; vehicle inspected by retailer.", Status: "Resolved", Campaigns: []string{"Sample campaign A"}, ServiceProviderName: "Example Jaguar North", ServiceProviderPostcode: "AB1 2CD", ServiceProviderAuthorised: sampleBool(true), FinalFixAt: "2025-11-20", DaysToFinalFix: sampleInt(6), CourtesyVehicleOffered: "Yes", CourtesyVehicleProvided: "Yes", PartsDelay: "up-to-1-week", GoodwillPayment: sampleBool(true), MilesDrivenWhilstFaulty: sampleInt(220), WarrantyCover: "Full", DisputeStatus: "None"},
		{VehicleID: "sample-vehicle-one", EventType: "service", OccurredAt: "2026-03-08", Mileage: sampleInt(44700), Title: "Scheduled service", Description: "Routine inspection and software updates.", Status: "Complete", ServiceProviderName: "Example Jaguar North", ServiceProviderPostcode: "AB1 2CD", ServiceProviderAuthorised: sampleBool(true), WarrantyCover: "Not applicable", DisputeStatus: "None"},
		{VehicleID: "sample-vehicle-two", EventType: "recall", OccurredAt: "2026-02-11", Mileage: sampleInt(17540), Title: "Recall inspection", Description: "Vehicle inspected and returned the same day.", Status: "Complete", Campaigns: []string{"Sample campaign B"}, ServiceProviderName: "Example Jaguar South", ServiceProviderPostcode: "XY9 8ZZ", ServiceProviderAuthorised: sampleBool(true), FinalFixAt: "2026-02-11", DaysToFinalFix: sampleInt(0), CourtesyVehicleOffered: "No", CourtesyVehicleProvided: "No", PartsDelay: "none", GoodwillPayment: sampleBool(false), MilesDrivenWhilstFaulty: sampleInt(0), WarrantyCover: "Full", DisputeStatus: "None"},
		{VehicleID: "sample-vehicle-two", EventType: "inspection", OccurredAt: "2026-07-22", Mileage: sampleInt(21790), Title: "Battery health check", Description: "Diagnostic report added to personal records.", Status: "Complete", WarrantyCover: "Not applicable", DisputeStatus: "None"},
	} {
		event.ID = "sample-event-" + string(rune('a'+index))
		event.CreatedAt = generated
		event.UpdatedAt = generated
		snapshot.ServiceEvents = append(snapshot.ServiceEvents, event)
	}

	body, err := buildMemberWorkbook(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("../../public/downloads/sample-ipace-owner-data.xlsx", body, 0o644); err != nil {
		t.Fatal(err)
	}
}
