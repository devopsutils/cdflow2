package terraform_test

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/test"
)

func TestTerraformInitInitial(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	// When
	if err := terraform.InitInitial(
		dockerClient,
		test.GetConfig("TEST_TERRAFORM_IMAGE"),
		test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
		buildVolume,
		&outputBuffer,
		&errorBuffer,
	); err != nil {
		log.Fatalln("unexpected error: ", err)
	}

	// Then
	if outputBuffer.String() != "message to stdout\n" {
		log.Fatalf("unexpected stdout output: '%v'", outputBuffer.String())
	}

	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		log.Panicln("error getting debug info:", err)
	}

	test.CheckTerraformInitInitialReflectedInput(debugInfo["terraform"])

	buildOutput, err := test.ReadVolume(dockerClient, buildVolume)
	if err != nil {
		log.Panicln("could not read build volume:", err)
	}

	if !reflect.DeepEqual(buildOutput, map[string][]byte{"build-output-test": []byte("build output")}) {
		log.Panicln("unexpected build output:", buildOutput)
	}
}

func TestTerraformConfigureBackend(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	// When
	func() {
		terraformContainer, err := terraform.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TERRAFORM_IMAGE"),
			test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
			releaseVolume,
		)
		if err != nil {
			log.Fatalln("error creating terraform container:", err)
		}
		defer func() {
			if err := terraformContainer.Done(); err != nil {
				log.Panicln("error cleaning up terraform container:", err)
			}
		}()

		if err := terraformContainer.ConfigureBackend(
			&outputBuffer,
			&errorBuffer,
			[]terraform.BackendConfigParameter{
				{"key1", "value1"},
				{"key2", "value2"},
			},
		); err != nil {
			log.Panicln("unexpected error: ", err)
		}
	}()

	// Then
	if outputBuffer.String() != "message to stdout\n" {
		log.Panicf("unexpected stdout output: '%v'", outputBuffer.String())
	}

	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		log.Panicln("error getting debug info:", err)
	}

	var input test.ReflectedInput
	if err := json.Unmarshal(debugInfo["terraform"], &input); err != nil {
		log.Panicln("error parsing json:", err)
	}
	if !reflect.DeepEqual(input.Args, []string{
		"init",
		"-get=false",
		"-get-plugins=false",
		"-backend-config=key1=value1",
		"-backend-config=key2=value2",
	}) {
		log.Panicln("unexpected args:", input.Args)
	}
}

func TestSwitchWorkspaceExisting(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	workspaceName := "existing-workspace"

	// When
	func() {
		terraformContainer, err := terraform.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TERRAFORM_IMAGE"),
			test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
			releaseVolume,
		)
		if err != nil {
			log.Fatalln("error creating terraform container:", err)
		}

		defer func() {
			if err := terraformContainer.Done(); err != nil {
				log.Panicln("error cleaning up terraform container:", err)
			}
		}()

		if err := terraformContainer.SwitchWorkspace(
			workspaceName,
			&outputBuffer,
			&errorBuffer,
		); err != nil {
			log.Panicln("error switching workspace:", err)
		}
	}()

	// Then
	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		log.Panicln("error getting debug info:", err)
	}

	lines := bytes.Split(debugInfo["terraform"], []byte{'\n'})
	if len(lines) != 3 || len(lines[2]) != 0 {
		log.Panicf("expected two lines with a trailing newline (empty string), got lines:\n%v", test.DumpLines(lines))
	}

	var listInput test.ReflectedInput
	if err := json.Unmarshal(lines[0], &listInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(listInput.Args, []string{"workspace", "list"}) {
		log.Panicln("unexpected args for workspace list:", listInput.Args)
	}

	var selectInput test.ReflectedInput
	if err := json.Unmarshal(lines[1], &selectInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(selectInput.Args, []string{"workspace", "select", workspaceName}) {
		log.Panicln("unexpected args for workspace select:", selectInput.Args)
	}
}

func TestSwitchWorkspaceNew(t *testing.T) {
	// When
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	workspaceName := "new-workspace"

	// When
	func() {
		terraformContainer, err := terraform.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TERRAFORM_IMAGE"),
			test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
			releaseVolume,
		)
		if err != nil {
			log.Fatalln("error creating terraform container:", err)
		}
		defer func() {
			if err := terraformContainer.Done(); err != nil {
				log.Panicln("error cleaning up terraform container:", err)
			}
		}()

		if err := terraformContainer.SwitchWorkspace(
			workspaceName,
			&outputBuffer,
			&errorBuffer,
		); err != nil {
			log.Panicln("error switching workspace:", err)
		}
	}()

	// Then
	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		log.Panicln("error getting debug info:", err)
	}

	lines := bytes.Split(debugInfo["terraform"], []byte{'\n'})
	if len(lines) != 3 || len(lines[2]) != 0 {
		log.Panicf("expected two lines with a trailing newline (empty string), got lines:\n%v", test.DumpLines(lines))
	}

	test.CheckTerraformWorkspaceList(lines[0])

	test.CheckTerraformWorkspaceNew(lines[1], workspaceName)
}
