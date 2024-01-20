package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rivo/tview"
	"golang.design/x/clipboard"
)

type BackEndType int

const (
	BACKEND_XDIS BackEndType = iota
	BACKEND_NATIVE
)

type PythonVersion int

const (
	PYTHON_OTHER PythonVersion = 0
	PYTHON_3_11  PythonVersion = 311
	PYTHON_3_12  PythonVersion = 312
)

func get_codeobject_list(backendType BackEndType, pythonVersion PythonVersion, pycFile string, pycMagic int) []string {
	var pythonInterp string
	switch pythonVersion {
	case PYTHON_3_12:
		pythonInterp = "lib\\python\\3.12\\python.exe"
	case PYTHON_3_11:
		fallthrough
	case PYTHON_OTHER:
		pythonInterp = "lib\\python\\3.11\\python.exe"
	}

	var backendStr string
	switch backendType {
	case BACKEND_NATIVE:
		backendStr = "native"
	case BACKEND_XDIS:
		backendStr = "xdis"
	}

	cmd := exec.Command(pythonInterp, "lib\\helper.py", "list", "--magic", strconv.Itoa(pycMagic), "--backend", backendStr, pycFile)

	output, err := cmd.Output()
	if err == nil {
		return strings.Split(strings.TrimSpace(string(output)), "\r\n")
	}
	return []string{}
}

func getCodeBytes(code string) string {
	var instruction_bytes string
	lines := strings.Split(code, "\r\n")

	for _, line := range lines {
		if strings.Count(line, "|") >= 2 {
			// Extract the text between the two |XX XX|
			instruction_bytes += strings.ReplaceAll(strings.SplitN(line, "|", 3)[1], " ", "")
		}
	}
	return instruction_bytes
}

func decompile(filename string) string {
	cmd := exec.Command("lib\\pycdc.exe", filename)
	output, err := cmd.CombinedOutput()
	if err == nil {
		txt := string(output)
		splitted := strings.SplitN(txt, "\r\n", 4)
		if strings.HasPrefix(splitted[0], "# Source Generated with Decompyle++") &&
			strings.HasPrefix(splitted[1], "# File: ") && splitted[2] == "" {
			return splitted[3]
		}
		return txt
	}
	return fmt.Sprintf("Error: %s", err)
}

func build_pyc(backendType BackEndType, pythonVersion PythonVersion, pycFile, outFile string, magic int, co_index int, codeBytes string) bool {
	var pythonInterp string
	switch pythonVersion {
	case PYTHON_3_12:
		pythonInterp = "lib\\python\\3.12\\python.exe"
	case PYTHON_3_11:
		fallthrough
	case PYTHON_OTHER:
		pythonInterp = "lib\\python\\3.11\\python.exe"
	}

	var backendStr string
	switch backendType {
	case BACKEND_NATIVE:
		backendStr = "native"
	case BACKEND_XDIS:
		backendStr = "xdis"
	}

	cmd := exec.Command(pythonInterp, "lib\\helper.py", "build", "--magic", strconv.Itoa(magic), "--backend", backendStr, "--index", strconv.Itoa(co_index), pycFile, outFile, codeBytes)

	_, err := cmd.Output()
	return err == nil
}

func build_UI(backendType BackEndType, pythonVersion PythonVersion, pycFile string, magic int, codeObject_list []string) {
	var app = tview.NewApplication()

	code_objects_list := tview.NewList()

	for _, co := range codeObject_list {
		coname_size := strings.Split(co, ":")
		code_objects_list.AddItem(coname_size[0], "size: "+coname_size[1], '.', nil)
	}

	left_flex := tview.NewFlex().AddItem(code_objects_list, 0, 1, false)
	left_flex.SetBorder(true).SetTitle("Code Objects")

	byteCode_textview := tview.NewTextArea()

	byteCode_textview.SetBorder(true).SetTitle("Bytecode (Input)")

	decompiledCode_textview := tview.NewTextView()
	decompiledCode_textview.SetBorder(true).SetTitle("Decompiled code (output)")

	clear_button := tview.NewButton("Clear")
	clear_button.SetBorder(true)
	clear_button.SetSelectedFunc(func() {
		byteCode_textview.SetText("", true)
		decompiledCode_textview.Clear()
	})

	paste_input_button := tview.NewButton("Paste Input")
	paste_input_button.SetBorder(true)
	paste_input_button.SetSelectedFunc(func() {
		clipdata := clipboard.Read(clipboard.FmtText)
		if clipdata != nil {
			byteCode_textview.SetText(string(clipdata), true)
			decompiledCode_textview.SetText("")
		}
	})

	copy_output_button := tview.NewButton("Copy Output")
	copy_output_button.SetBorder(true)
	copy_output_button.SetSelectedFunc(func() {
		output := strings.TrimSpace(decompiledCode_textview.GetText(false))
		clipboard.Write(clipboard.FmtText, []byte(output))
	})

	decompile_button := tview.NewButton("Decompile")
	decompile_button.SetBorder(true)
	decompile_button.SetSelectedFunc(func() {
		code := byteCode_textview.GetText()
		if len(code) > 0 {
			codeBytes := getCodeBytes(code)
			co_index := code_objects_list.GetCurrentItem()

			tf, _ := os.CreateTemp("lib", "temp*.pyc")
			tf.Close()
			if build_pyc(backendType, pythonVersion, pycFile, tf.Name(), magic, co_index, codeBytes) {
				output := decompile(tf.Name())
				decompiledCode_textview.SetText(output)
			}
			os.Remove(tf.Name())
		}
	})

	quit_button := tview.NewButton("Quit")
	quit_button.SetBorder(true)
	quit_button.SetSelectedFunc(func() {
		app.Stop()
	})

	button_panel := tview.NewFlex().SetDirection(tview.FlexColumn)
	button_panel.AddItem(clear_button, 0, 1, false)
	button_panel.AddItem(paste_input_button, 0, 1, false)
	button_panel.AddItem(copy_output_button, 0, 1, false)
	button_panel.AddItem(decompile_button, 0, 1, false)
	button_panel.AddItem(quit_button, 0, 1, false)
	button_panel.SetBorderPadding(0, 1, 0, 0)

	right_flex := tview.NewFlex().SetDirection(tview.FlexRow)
	right_flex.AddItem(byteCode_textview, 0, 7, false)
	right_flex.AddItem(decompiledCode_textview, 0, 7, false)
	right_flex.AddItem(button_panel, 0, 1, false)

	main_flex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(left_flex, 0, 1, true).
		AddItem(right_flex, 0, 3, false)

	if err := app.SetRoot(main_flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func getPycMagic(pycFile string) int {
	contents, err := os.ReadFile(pycFile)
	if err == nil {
		return int(contents[1])<<8 | int(contents[0])
	}
	return -1
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <pycfile>\n", os.Args[0])
		return
	}
	pycFile := os.Args[1]

	pycMagic := getPycMagic(pycFile)
	var backendType BackEndType
	var pythonVersion PythonVersion
	switch pycMagic {
	case 0x0DCB: // Python 3.12
		backendType = BACKEND_NATIVE
		pythonVersion = PYTHON_3_12
	case 0x0DA7: // Python 3.11
		backendType = BACKEND_NATIVE
		pythonVersion = PYTHON_3_11
	default:
		pythonVersion = PYTHON_OTHER
		backendType = BACKEND_XDIS
	}

	codeObject_list := get_codeobject_list(backendType, pythonVersion, pycFile, pycMagic)

	if len(codeObject_list) > 0 {
		build_UI(backendType, pythonVersion, pycFile, pycMagic, codeObject_list)
	}
}
