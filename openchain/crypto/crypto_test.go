/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package crypto

import (
	pb "github.com/openblockchain/obc-peer/protos"

	"fmt"
	"github.com/openblockchain/obc-peer/obcca/obcca"
	"github.com/openblockchain/obc-peer/openchain/crypto/utils"
	"github.com/openblockchain/obc-peer/openchain/util"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

var (
	validatorConf utils.NodeConfiguration
	validator     Peer

	deployer Client
	invoker  Client

	caAlreadyOn bool
	eca         *obcca.ECA
	tca         *obcca.TCA
	caWaitGroup sync.WaitGroup
)

func TestMain(m *testing.M) {
	setupTestConfig()

	// Init ECA
	go initMockCAs()
	defer cleanup()

	// Init a mock Client
	err := initClients()
	if err != nil {
		fmt.Printf("Failed initializing clients: %s\n", err)
		panic(fmt.Errorf("Failed initializing clients: %s", err))
	}

	// Register
	validatorConf = utils.NodeConfiguration{Type: "validator", Name: "validator"}
	err = RegisterValidator(validatorConf.Name, nil, validatorConf.GetEnrollmentID(), validatorConf.GetEnrollmentPWD())
	if err != nil {
		fmt.Printf("Failed registerting: %s\n", err)
		killCAs()
		panic(fmt.Errorf("Failed registerting: %s", err))
	}

	//	 Verify that a second call to Register fails
	err = RegisterValidator(validatorConf.Name, nil, validatorConf.GetEnrollmentID(), validatorConf.GetEnrollmentPWD())
	if err != nil {
		fmt.Printf("Failed checking registerting: %s\n", err)
		killCAs()
		panic(fmt.Errorf("Failed checking registration: %s", err))
	}

	// Init client
	validator, err = InitValidator(validatorConf.Name, nil)

	var ret int
	if err != nil {
		panic(fmt.Errorf("Failed initializing: err %s", err))
	} else {
		ret = m.Run()
	}

	cleanup()

	os.Exit(ret)
}

func TestRegistration(t *testing.T) {
	err := RegisterValidator(validatorConf.Name, nil, validatorConf.GetEnrollmentID(), validatorConf.GetEnrollmentPWD())

	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestID(t *testing.T) {
	// Verify that any id modification doesn't change
	id := validator.GetID()

	if id == nil {
		t.Fatalf("Id is nil.")
	}

	if len(id) == 0 {
		t.Fatalf("Id length is zero.")
	}

	id[0] = id[0] + 1
	id2 := validator.GetID()
	if id2[0] == id[0] {
		t.Fatalf("Invariant not respected.")
	}
}

func TestDeployTransactionPreValidation(t *testing.T) {
	tx, err := mockDeployTransaction()
	if err != nil {
		t.Fatalf("TransactionPreValidation: failed creating transaction: %s", err)
	}

	res, err := validator.TransactionPreValidation(tx)
	if res == nil {
		t.Fatalf("TransactionPreValidation: result must be diffrent from nil: %s", err)
	}
	if err != nil {
		t.Fatalf("TransactionPreValidation: failed pre validing transaction: %s", err)
	}
}

func TestInvokeTransactionPreValidation(t *testing.T) {
	tx, err := mockInvokeTransaction()
	if err != nil {
		t.Fatalf("TransactionPreValidation: failed creating transaction: %s", err)
	}

	res, err := validator.TransactionPreValidation(tx)
	if res == nil {
		t.Fatalf("TransactionPreValidation: result must be diffrent from nil")
	}
	if err != nil {
		t.Fatalf("TransactionPreValidation: failed pre validing transaction: %s", err)
	}
}

func TestDeployTransactionPreExecution(t *testing.T) {
	tx, err := mockDeployTransaction()
	if err != nil {
		t.Fatalf("TransactionPreExecution: failed creating transaction: %s", err)
	}

	res, err := validator.TransactionPreExecution(tx)
	if res == nil {
		t.Fatalf("TransactionPreExecution: result must be diffrent from nil")
	}
	if err != nil {
		t.Fatalf("TransactionPreExecution: failed pre validing transaction: %s", err)
	}
}

func TestInvokeTransactionPreExecution(t *testing.T) {
	tx, err := mockInvokeTransaction()
	if err != nil {
		t.Fatalf("TransactionPreExecution: failed creating transaction: %s", err)
	}

	res, err := validator.TransactionPreExecution(tx)
	if res == nil {
		t.Fatalf("TransactionPreExecution: result must be diffrent from nil")
	}
	if err != nil {
		t.Fatalf("TransactionPreExecution: failed pre validing transaction: %s", err)
	}
}

func TestSignVerify(t *testing.T) {
	msg := []byte("Hello World!!!")
	signature, err := validator.Sign(msg)
	if err != nil {
		t.Fatalf("TestSign: failed generating signature: %s", err)
	}

	err = validator.Verify(validator.GetID(), signature, msg)
	if err != nil {
		t.Fatalf("TestSign: failed validating signature: %s", err)
	}
}

func Test_NewChaincodeDeployTransaction(t *testing.T) {
	uuid, err := util.GenerateUUID()
	if err != nil {
		t.Fatalf("Test_NewChaincodeDeployTransaction: failed generating uuid: err %s", err)
	}
	tx, err := deployer.NewChaincodeDeployTransaction(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				Type:        pb.ChaincodeSpec_GOLANG,
				ChaincodeID: &pb.ChaincodeID{Url: "Contract001", Version: "0.0.1"},
				CtorMsg:     nil,
			},
			EffectiveDate: nil,
			CodePackage:   nil,
		},
		uuid,
	)

	if err != nil {
		t.Fatalf("Test_NewChaincodeDeployTransaction: failed creating NewChaincodeDeployTransaction: err %s", err)
	}

	if tx == nil {
		t.Fatalf("Test_NewChaincodeDeployTransaction: failed creating NewChaincodeDeployTransaction: result is nil")
	}

	// Check transaction
	//	err = client.(*node.clientImpl).checkTransaction(tx)
	//	if err != nil {
	//		t.Fatalf("Test_NewChaincodeDeployTransaction: failed checking transaction: err %s", err)
	//	}
}

func Test_NewChaincodeInvokeTransaction(t *testing.T) {
	uuid, err := util.GenerateUUID()
	if err != nil {
		t.Fatalf("Test_NewChaincodeInvokeTransaction: failed generating uuid: err %s", err)
	}
	tx, err := deployer.NewChaincodeExecute(
		&pb.ChaincodeInvocationSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				Type:        pb.ChaincodeSpec_GOLANG,
				ChaincodeID: &pb.ChaincodeID{Url: "Contract001", Version: "0.0.1"},
				CtorMsg:     nil,
			},
		},
		uuid,
	)

	if err != nil {
		t.Fatalf("Test_NewChaincodeInvokeTransaction: failed creating NewChaincodeInvokeTransaction: err %s", err)
	}

	if tx == nil {
		t.Fatalf("Test_NewChaincodeInvokeTransaction: failed creating NewChaincodeInvokeTransaction: result is nil")
	}

	// TODO
	//	err = client.checkTransaction(tx)
	//	if err != nil {
	//		t.Fatalf("Test_NewChaincodeInvokeTransaction: failed checking transaction: err %s", err)
	//	}
}

func Test_MultipleNewChaincodeInvokeTransaction(t *testing.T) {
	for i := 0; i < 24; i++ {
		uuid, err := util.GenerateUUID()
		if err != nil {
			t.Fatalf("Test_MultipleNewChaincodeInvokeTransaction: failed generating uuid: err %s", err)
		}
		tx, err := deployer.NewChaincodeExecute(
			&pb.ChaincodeInvocationSpec{
				ChaincodeSpec: &pb.ChaincodeSpec{
					Type:        pb.ChaincodeSpec_GOLANG,
					ChaincodeID: &pb.ChaincodeID{Url: "Contract001", Version: "0.0.1"},
					CtorMsg:     nil,
				},
			},
			uuid,
		)

		if err != nil {
			t.Fatalf("Test_MultipleNewChaincodeInvokeTransaction: failed creating NewChaincodeInvokeTransaction: err %s", err)
		}

		if tx == nil {
			t.Fatalf("Test_MultipleNewChaincodeInvokeTransaction: failed creating NewChaincodeInvokeTransaction: result is nil")
		}

		//		TODO
		//		err = client.checkTransaction(tx)
		//		if err != nil {
		//			t.Fatalf("Test_MultipleNewChaincodeInvokeTransaction: failed checking transaction: err %s", err)
		//		}

	}
}

func setupTestConfig() {
	viper.SetConfigName("crypto_test") // name of config file (without extension)
	viper.AddConfigPath(".")           // path to look for the config file in
	err := viper.ReadInConfig()        // Find and read the config file
	if err != nil {                    // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	removeFolders()
}

func initMockCAs() {
	// Check if the CAs are already up
	if err := utils.IsTCPPortOpen(viper.GetString("ports.ecaP")); err != nil {
		caAlreadyOn = true
		fmt.Println("Someone already listening")
		return
	}
	caAlreadyOn = false

	obcca.LogInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr, os.Stdout)

	eca = obcca.NewECA()
	defer eca.Close()
	eca.Start(&caWaitGroup)

	tca = obcca.NewTCA(eca)
	defer tca.Close()
	tca.Start(&caWaitGroup)

	caWaitGroup.Wait()
}

func initClients() error {
	// Deployer
	deployerConf := utils.NodeConfiguration{Type: "client", Name: "user4"}
	if err := RegisterClient(deployerConf.Name, nil, deployerConf.GetEnrollmentID(), deployerConf.GetEnrollmentPWD()); err != nil {
		return err
	}
	var err error
	deployer, err = InitClient(deployerConf.Name, nil)
	if err != nil {
		return err
	}

	// Invoker
	invokerConf := utils.NodeConfiguration{Type: "client", Name: "user5"}
	if err := RegisterClient(invokerConf.Name, nil, invokerConf.GetEnrollmentID(), invokerConf.GetEnrollmentPWD()); err != nil {
		return err
	}
	invoker, err = InitClient(invokerConf.Name, nil)
	if err != nil {
		return err
	}

	return nil
}

func mockDeployTransaction() (*pb.Transaction, error) {
	tx, err := deployer.NewChaincodeDeployTransaction(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				Type:        pb.ChaincodeSpec_GOLANG,
				ChaincodeID: &pb.ChaincodeID{Url: "Contract001", Version: "0.0.1"},
				CtorMsg:     nil,
			},
			EffectiveDate: nil,
			CodePackage:   nil,
		},
		"uuid",
	)
	return tx, err
}

func mockInvokeTransaction() (*pb.Transaction, error) {
	tx, err := invoker.NewChaincodeExecute(
		&pb.ChaincodeInvocationSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				Type:        pb.ChaincodeSpec_GOLANG,
				ChaincodeID: &pb.ChaincodeID{Url: "Contract001", Version: "0.0.1"},
				CtorMsg:     nil,
			},
		},
		"uuid",
	)

	return tx, err
}

func cleanup() {
	CloseAllClients()
	CloseAllValidators()
	killCAs()

	fmt.Println("Prepare to cleanup...")
	//	time.Sleep(40 * time.Second)

	fmt.Println("Test...")
	if err := utils.IsTCPPortOpen(viper.GetString("ports.ecaP")); err != nil {
		fmt.Println("AAA Someone already listening")
	}
	removeFolders()
	fmt.Println("Cleanup...done!")
}

func killCAs() {
	if !caAlreadyOn {
		eca.Stop()
		eca.Close()

		tca.Stop()
		tca.Close()
	}
}

func removeFolders() {
	if err := os.RemoveAll(viper.GetString("peer.fileSystemPath")); err != nil {
		fmt.Printf("Failed removing [%s]: %s\n", viper.GetString("peer.fileSystemPath"), err)
	}
	if err := os.RemoveAll(viper.GetString("eca.crypto.path")); err != nil {
		fmt.Printf("Failed removing [%s]: %s\n", viper.GetString("eca.crypto.path"), err)
	}
}