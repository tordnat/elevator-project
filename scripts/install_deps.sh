#!/bin/bash

HRA_INSTALL_PATH="elevatorAlgorithm/hra/"
echo "Building project and moving hall_request_assigner to $HRA_INSTALL_PATH"

WORKING_DIR=$(pwd)

sudo chmod +x Project-resources/cost_fns/hall_request_assigner/build.sh
cd Project-resources/cost_fns/hall_request_assigner/
./build.sh
cd $WORKING_DIR
sudo mv Project-resources/cost_fns/hall_request_assigner $HRA_INSTALL_PATH
sudo chmod +x $HRA_INSTALL_PATH/hall_request_assigner
