package ipace

import "time"

type joinRequest struct {
	BotField        string      `json:"bot-field"`
	Name            string      `json:"name"`
	Email           string      `json:"email"`
	Country         string      `json:"country"`
	Relationship    string      `json:"relationship"`
	Skills          stringArray `json:"skills"`
	ConsentContact  string      `json:"consent-contact"`
	ConsentNotLegal string      `json:"consent-not-legal"`
	ConsentData     string      `json:"consent-data"`
}

type vehicleRequest struct {
	VIN          string `json:"vin"`
	Registration string `json:"registration"`
	Country      string `json:"country"`
	ModelYear    string `json:"modelYear"`
	Mileage      string `json:"mileage"`
	OwnedSince   string `json:"ownedSince"`
	FirstReg     string `json:"firstReg"`
	SOH          string `json:"soh"`
	SOHDate      string `json:"sohDate"`
	SOHMileage   string `json:"sohMileage"`
	SOHSource    string `json:"sohSource"`
}

type batteryReadingRequest struct {
	VehicleID  string `json:"vehicleId"`
	SOH        string `json:"soh"`
	SOHDate    string `json:"sohDate"`
	SOHMileage string `json:"sohMileage"`
	SOHSource  string `json:"sohSource"`
}

type serviceEventRequest struct {
	ID                      string      `json:"id"`
	VehicleID               string      `json:"vehicleId"`
	EventType               string      `json:"eventType"`
	OccurredAt              string      `json:"occurredAt"`
	Mileage                 string      `json:"mileage"`
	Title                   string      `json:"title"`
	Description             string      `json:"description"`
	Status                  string      `json:"status"`
	Campaigns               stringArray `json:"campaigns"`
	FinalFixAt              string      `json:"finalFixAt"`
	DaysToFinalFix          string      `json:"daysToFinalFix"`
	CourtesyVehicleOffered  string      `json:"courtesyVehicleOffered"`
	CourtesyVehicleProvided string      `json:"courtesyVehicleProvided"`
	PartsDelay              string      `json:"partsDelay"`
	WarrantyCover           string      `json:"warrantyCover"`
	DisputeStatus           string      `json:"disputeStatus"`
}

type magicLinkRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type contactRecord struct {
	Name    string `json:"name,omitempty" firestore:"name,omitempty"`
	Email   string `json:"email,omitempty" firestore:"email,omitempty"`
	Country string `json:"country,omitempty" firestore:"country,omitempty"`
}

type membershipRecord struct {
	Relationship string   `json:"relationship,omitempty" firestore:"relationship,omitempty"`
	Skills       []string `json:"skills,omitempty" firestore:"skills,omitempty"`
}

type consentRecord struct {
	Contact            bool `json:"contact" firestore:"contact"`
	NotLegalClaim      bool `json:"notLegalClaim" firestore:"notLegalClaim"`
	AnonymisedAnalysis bool `json:"anonymisedAnalysis" firestore:"anonymisedAnalysis"`
}

type vehicleDetails struct {
	VINHash               string `json:"vinHash,omitempty" firestore:"vinHash,omitempty"`
	VINLast6              string `json:"vinLast6,omitempty" firestore:"vinLast6,omitempty"`
	Registration          string `json:"registration,omitempty" firestore:"registration,omitempty"`
	Country               string `json:"country,omitempty" firestore:"country,omitempty"`
	ModelYear             string `json:"modelYear,omitempty" firestore:"modelYear,omitempty"`
	Mileage               *int   `json:"mileage,omitempty" firestore:"mileage,omitempty"`
	OwnedSince            string `json:"ownedSince,omitempty" firestore:"ownedSince,omitempty"`
	FirstRegistrationDate string `json:"firstRegistrationDate,omitempty" firestore:"firstRegistrationDate,omitempty"`
}

type batteryDetails struct {
	StateOfHealth        *float64 `json:"stateOfHealth,omitempty" firestore:"stateOfHealth,omitempty"`
	MeasuredAt           string   `json:"measuredAt,omitempty" firestore:"measuredAt,omitempty"`
	MileageAtMeasurement *int     `json:"mileageAtMeasurement,omitempty" firestore:"mileageAtMeasurement,omitempty"`
	Source               string   `json:"source,omitempty" firestore:"source,omitempty"`
}

type reviewRecord struct {
	Status            string `json:"status" firestore:"status"`
	VerificationLevel string `json:"verificationLevel" firestore:"verificationLevel"`
}

type joinRecord struct {
	ID             string           `json:"id" firestore:"id"`
	Type           string           `json:"type" firestore:"type"`
	CreatedAt      time.Time        `json:"createdAt" firestore:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt" firestore:"updatedAt"`
	IdentityUserID string           `json:"identityUserId,omitempty" firestore:"identityUserId,omitempty"`
	UserEmailHash  string           `json:"userEmailHash" firestore:"userEmailHash"`
	Contact        contactRecord    `json:"contact" firestore:"contact"`
	Membership     membershipRecord `json:"membership" firestore:"membership"`
	Consents       consentRecord    `json:"consents" firestore:"consents"`
	Review         reviewRecord     `json:"review" firestore:"review"`
}

type vehicleRecord struct {
	ID             string         `json:"id" firestore:"id"`
	Type           string         `json:"type" firestore:"type"`
	CreatedAt      time.Time      `json:"createdAt" firestore:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt" firestore:"updatedAt"`
	IdentityUserID string         `json:"identityUserId" firestore:"identityUserId"`
	UserEmailHash  string         `json:"userEmailHash,omitempty" firestore:"userEmailHash,omitempty"`
	Vehicle        vehicleDetails `json:"vehicle" firestore:"vehicle"`
	Battery        batteryDetails `json:"battery" firestore:"battery"`
	Review         reviewRecord   `json:"review" firestore:"review"`
}

type batteryReadingRecord struct {
	ID             string         `json:"id" firestore:"id"`
	Type           string         `json:"type" firestore:"type"`
	CreatedAt      time.Time      `json:"createdAt" firestore:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt" firestore:"updatedAt"`
	IdentityUserID string         `json:"identityUserId" firestore:"identityUserId"`
	VehicleID      string         `json:"vehicleId" firestore:"vehicleId"`
	Battery        batteryDetails `json:"battery" firestore:"battery"`
	Review         reviewRecord   `json:"review" firestore:"review"`
}

type serviceEventRecord struct {
	ID                      string       `json:"id" firestore:"id"`
	Type                    string       `json:"type" firestore:"type"`
	CreatedAt               time.Time    `json:"createdAt" firestore:"createdAt"`
	UpdatedAt               time.Time    `json:"updatedAt" firestore:"updatedAt"`
	IdentityUserID          string       `json:"identityUserId" firestore:"identityUserId"`
	VehicleID               string       `json:"vehicleId" firestore:"vehicleId"`
	EventType               string       `json:"eventType" firestore:"eventType"`
	OccurredAt              string       `json:"occurredAt" firestore:"occurredAt"`
	Mileage                 *int         `json:"mileage,omitempty" firestore:"mileage,omitempty"`
	Title                   string       `json:"title" firestore:"title"`
	Description             string       `json:"description,omitempty" firestore:"description,omitempty"`
	Status                  string       `json:"status" firestore:"status"`
	Campaigns               []string     `json:"campaigns,omitempty" firestore:"campaigns,omitempty"`
	FinalFixAt              string       `json:"finalFixAt,omitempty" firestore:"finalFixAt,omitempty"`
	DaysToFinalFix          *int         `json:"daysToFinalFix,omitempty" firestore:"daysToFinalFix,omitempty"`
	CourtesyVehicleOffered  string       `json:"courtesyVehicleOffered,omitempty" firestore:"courtesyVehicleOffered,omitempty"`
	CourtesyVehicleProvided string       `json:"courtesyVehicleProvided,omitempty" firestore:"courtesyVehicleProvided,omitempty"`
	PartsDelay              string       `json:"partsDelay,omitempty" firestore:"partsDelay,omitempty"`
	WarrantyCover           string       `json:"warrantyCover,omitempty" firestore:"warrantyCover,omitempty"`
	DisputeStatus           string       `json:"disputeStatus,omitempty" firestore:"disputeStatus,omitempty"`
	Review                  reviewRecord `json:"review" firestore:"review"`
}

type memberSnapshot struct {
	IdentityUserID  string                 `json:"identityUserId" firestore:"identityUserId"`
	Email           string                 `json:"email,omitempty" firestore:"email,omitempty"`
	GeneratedAt     time.Time              `json:"generatedAt" firestore:"generatedAt"`
	JoinRecords     []joinRecord           `json:"joinRecords" firestore:"joinRecords"`
	VehicleRecords  []vehicleRecord        `json:"vehicleRecords" firestore:"vehicleRecords"`
	BatteryReadings []batteryReadingRecord `json:"batteryReadings" firestore:"batteryReadings"`
	ServiceEvents   []serviceEventRecord   `json:"serviceEvents" firestore:"serviceEvents"`
}

type adminData struct {
	JoinRecords    []joinRecord    `json:"joinRecords"`
	VehicleRecords []vehicleRecord `json:"vehicleRecords"`
}

type publicDistributionBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type publicStatsSnapshot struct {
	GeneratedAt           time.Time                  `json:"generatedAt"`
	OwnersContributed     int                        `json:"ownersContributed"`
	VehiclesRegistered    int                        `json:"vehiclesRegistered"`
	VehiclesWithSOH       int                        `json:"vehiclesWithSoh"`
	SOHReadings           int                        `json:"sohReadings"`
	VehiclesWithRepeatSOH int                        `json:"vehiclesWithRepeatSoh"`
	AverageReportedSOH    *float64                   `json:"averageReportedSoh,omitempty"`
	AverageSOHChange      *float64                   `json:"averageSohChange,omitempty"`
	SOHDistribution       []publicDistributionBucket `json:"sohDistribution"`
	ModelYearDistribution []publicDistributionBucket `json:"modelYearDistribution"`
}

type stringArray []string

func (s *stringArray) UnmarshalJSON(data []byte) error {
	var one string
	if err := jsonUnmarshal(data, &one); err == nil {
		if one == "" {
			*s = nil
		} else {
			*s = []string{one}
		}
		return nil
	}

	var many []string
	if err := jsonUnmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}
