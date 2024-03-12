#!/bin/bash

HRA_INSTALL_PATH="elevatorAlgorithm/hra/"
echo "Building project and moving hall_request_assigner to $HRA_INSTALL_PATH"

Project-resources/cost_fns/hall_request_assigner/build.sh
sudo cp Project-resources/cost_fns/hall_request_assigner $HRA_INSTALL_PATH