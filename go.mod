module github.com/fastly/compute-sdk-go

// NOTE: When updating the go line, update this special tinygo comment 
// to a version compatible with the go version:
//+tinygo 0.33.0
go 1.23.12

retract (
	v1.4.1 // Contains retractions only
	v1.4.0 // Observed errors after rollout
)
