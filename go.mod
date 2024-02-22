module elevator-project

require networkDriver v0.0.0

replace networkDriver => ./networkDriver

require elevatorDriver v0.0.0

replace elevatorDriver => ./elevatorDriver

require elevatorAlgorithm v0.0.0

replace elevatorAlgorithm => ./elevatorAlgorithm

require elevatorMusic v0.0.0

replace elevatorMusic => ./elevatorMusic

go 1.20
