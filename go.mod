module github.com/fastly/compute-sdk-go

go 1.21

retract (
	v1.4.0 // Observed errors after rollout
	v1.4.1 // Contains retractions only
)
