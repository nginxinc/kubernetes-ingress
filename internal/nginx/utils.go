package nginx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/golang/glog"
)

func shellOut(cmd string) (err error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	glog.V(3).Infof("executing %s", cmd)

	command := exec.Command("sh", "-c", cmd)
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Start()
	if err != nil {
		return fmt.Errorf("failed to execute %v, err: %w", cmd, err)
	}

	err = command.Wait()
	if err != nil {
		return fmt.Errorf("command %v stdout: %q\nstderr: %q\nfinished with error: %w", cmd,
			stdout.String(), stderr.String(), err)
	}
	return nil
}

func createFileAndWrite(name string, b []byte) error {
	w, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("failed to open %v: %w", name, err)
	}

	defer func() {
		if tempErr := w.Close(); tempErr != nil {
			err = tempErr
		}
	}()

	_, err = w.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write to %v: %w", name, err)
	}

	return err
}

func createFileAndWriteAtomically(filename string, tempPath string, mode os.FileMode, content []byte) {
	file, err := os.CreateTemp(tempPath, path.Base(filename))
	if err != nil {
		glog.Fatalf("Couldn't create a temp file for the file %v: %v", filename, err)
	}

	err = file.Chmod(mode)
	if err != nil {
		glog.Fatalf("Couldn't change the mode of the temp file %v: %v", file.Name(), err)
	}

	_, err = file.Write(content)
	if err != nil {
		glog.Fatalf("Couldn't write to the temp file %v: %v", file.Name(), err)
	}

	err = file.Close()
	if err != nil {
		glog.Fatalf("Couldn't close the temp file %v: %v", file.Name(), err)
	}

	err = os.Rename(file.Name(), filename)
	if err != nil {
		glog.Fatalf("Couldn't rename the temp file %v to %v: %v", file.Name(), filename, err)
	}
}
