module elevator-project

require networkDriver v0.0.0
replace networkDriver => ./networkDriver

require elevatorDriver v0.0.0
replace elevatorDriver => ./elevatorDriver
go 1.20