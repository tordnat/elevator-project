module elevator-project

require networkDriver v0.0.0

replace networkDriver => ./networkDriver

require elevatorDriver v0.0.0

replace elevatorDriver => ./elevatorDriver

require elevatorAlgorithm v0.0.0

require github.com/adrg/libvlc-go/v3 v3.1.5 // indirect

replace elevatorAlgorithm => ./elevatorAlgorithm

go 1.20
