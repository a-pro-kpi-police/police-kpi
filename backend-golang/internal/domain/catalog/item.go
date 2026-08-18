package catalog

type Item struct {
	ID            int
	Code          string
	Indicator     string
	Block         string
	Subblock      string
	NPA           string
	PointsPerUnit float64
	Unit          string
	SourceURL     string
	MonthlyLimit  float64
	IsMass        bool
	IsCritical    bool
	IsActive      bool
}
