package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viduq/vv104"

	"github.com/AllenDang/giu"
	"github.com/sqweek/dialog"
	"golang.org/x/image/colornames"
)

var (
	sashPos1        float32  = 100
	sashPos2        float32  = 250
	colorConnection          = colornames.Cadetblue
	typeIds         []string = vv104.TypeIDs
	typeIdSelected  int32
	causeTxs        []string = vv104.CauseTxs
	causeTxSelected int32
	typeIdStr       string
	valueStr        string = "0"
	value           vv104.InfoValue

	moniObjectSelected int32
	ctrlObjectSelected int32
	casdu              int16  = 1
	casduStr           string = "1"

	ioa            int32 = 1
	objName        string
	ipAddrStr      string = "127.0.0.1"
	portStr        string = "2404"
	mode           string = "client"
	autoScrolling  bool   = true
	errorTextIp    string = ""
	errorTextPort  string = ""
	errorTextCasdu string = ""
	k              string = "12"
	w              string = "8"
	t1             string = "15"
	t2             string = "10"
	t3             string = "20"

	state vv104.State
	// objects vv104.Objects

	ctx     context.Context
	cancel  context.CancelFunc
	logChan <-chan string
	logTxt  string

	wg sync.WaitGroup
)

func loop() {
	wg.Add(1)
	defer wg.Done()

	giu.SingleWindowWithMenuBar().Layout(
		giu.MenuBar().Layout(
			giu.Menu("File").Layout(
				giu.MenuItem("Open Config").Shortcut("Ctrl+o").OnClick(openConfig),
				giu.MenuItem("Save Config").Shortcut("Ctrl+s").OnClick(saveConfig),
			),
		),
		giu.SplitLayout(giu.DirectionHorizontal, &sashPos1,
			giu.Layout{
				giu.Label("Configuration"),
				giu.Column(
					giu.Row(
						giu.RadioButton("Client", mode == "client").OnChange(func() { mode = "client" }),
						giu.RadioButton("Server", mode == "server").OnChange(func() { mode = "server" }),
						giu.Label("IP:"),
						giu.InputText(&ipAddrStr).OnChange(checkIpAddr).Label("###ipInput").Size(150),
						giu.Label(errorTextIp),
						giu.Label("Port:"),
						giu.InputText(&portStr).Label("###portInput").Size(50).Flags(giu.InputTextFlagsCharsDecimal).OnChange(checkPort),
						giu.Label(errorTextPort),

						giu.Label("Casdu:"),
						giu.InputText(&casduStr).Label("###casduInput").Size(50).OnChange(checkCasdu),
						giu.Label(errorTextCasdu),

						giu.Label("k:"),
						giu.InputText(&k).Label("###k").Size(50).Flags(giu.InputTextFlagsCharsDecimal),
						giu.Label("w:"),
						giu.InputText(&w).Label("###w").Size(50).Flags(giu.InputTextFlagsCharsDecimal),

						giu.Label("t1:"),
						giu.InputText(&t1).Label("###t1").Size(50).Flags(giu.InputTextFlagsCharsDecimal),
						giu.Label("t2:"),
						giu.InputText(&t2).Label("###t2").Size(50).Flags(giu.InputTextFlagsCharsDecimal),
						giu.Label("t3:"),
						giu.InputText(&t3).Label("###t3").Size(50).Flags(giu.InputTextFlagsCharsDecimal),
					),
					giu.Row(
						giu.Style().SetColor(giu.StyleColorButton, colorConnection).To(giu.Button("Connect").OnClick(connectIec104).Disabled(state.Running)),
						giu.Button("Disconnect").OnClick(disconnectIec104).Disabled(!state.Running),
						// giu.Style().SetColor(giu.StyleColorChildBg, colornames.Deeppink).To(giu.Child().Size(50, 30)),
						// giu.Style().SetColor(giu.StyleColorChildBg, colorConnection).To(giu.Child().Size(50, 30)),
					),
				),
			},
			giu.SplitLayout(giu.DirectionVertical, &sashPos2,
				giu.Layout{giu.Label("Objects"),
					giu.Column(
						giu.Combo("Type ID", typeIds[typeIdSelected], typeIds, &typeIdSelected),
						giu.Label("IOA:"),
						giu.InputInt(&ioa),
						giu.Label("Obj. Name:"),
						giu.InputText(&objName),
					),
					giu.Row(
						giu.Button("Add object").OnClick(addObject),
					),
					giu.Column(
						giu.TabBar().TabItems(
							giu.TabItem("Monitoring").Layout(
								giu.ListBox("Monitoring", state.Objects.MoniList).SelectedIndex(&moniObjectSelected).Size(sashPos2, 300),
								giu.Button("Remove Moni Obj.").OnClick(removeMoniObject),
								giu.Button("Send M. Obj.").OnClick(sendMoniObject).Size(100, 30),
							),
							giu.TabItem("Control").Layout(
								giu.ListBox("Control", state.Objects.CtrlList).SelectedIndex(&ctrlObjectSelected).Size(sashPos2, 300),
								giu.Button("Remove Ctrl Obj.").OnClick(removeCtrlObject),
								giu.Button("Send C. Obj.").OnClick(sendCtrlObject).Size(100, 30),
							),
							giu.TabItem("Special").Layout(
								giu.Row(
									giu.Button("Send StartDT act").OnClick(func() { state.Chans.CommandsFromStdin <- "startdt_act" }),
									giu.Button("Send StartDT con").OnClick(func() { state.Chans.CommandsFromStdin <- "startdt_con" }),
								),
								giu.Row(
									giu.Button("Send StopDT act").OnClick(func() { state.Chans.CommandsFromStdin <- "stopdt_act" }),
									giu.Button("Send StopDT con").OnClick(func() { state.Chans.CommandsFromStdin <- "stopdt_con" }),
								),
								giu.Row(
									giu.Button("Send TestFr act").OnClick(func() { state.Chans.CommandsFromStdin <- "testfr_act" }),
									giu.Button("Send TestFr con").OnClick(func() { state.Chans.CommandsFromStdin <- "testfr_con" }),
								),
								giu.Row(
									giu.Button("Send GI cmd").OnClick(sendGI),
								),
							),
						),
						giu.Combo("Cause Tx", causeTxs[causeTxSelected], causeTxs, &causeTxSelected),
						giu.InputText(&valueStr),
					),
				},
				giu.Layout{
					giu.Row(
						giu.Checkbox("Auto Scrolling", &autoScrolling),
						giu.Button("Clear").OnClick(func() { logTxt = "" }),
						giu.Button("Copy").OnClick(func() { logTxt = "" }),
					),
					// giu.Label("Log"),
					giu.InputTextMultiline(&logTxt).AutoScrollToBottom(autoScrolling).Flags(giu.InputTextFlagsReadOnly).Size(600, 600),
					// giu.ListBox("LogBox", logTxt).SelectedIndex(&selectedIndex),
					// giu.Table().Columns(giu.TableColumn("")).Rows(lines...),
				}),
		),
	)

	if state.TcpConnected {
		colorConnection = colornames.Lightgreen
	} else {
		colorConnection = colornames.Cadetblue
	}

}

func main() {
	wnd := giu.NewMasterWindow("Brez", 1000, 800, 0)
	state = vv104.NewState()

	wnd.Run(loop)
}

func connectIec104() {

	ctx, cancel = context.WithCancel(context.Background())

	writeConfigsToState()

	if state.Config.LogToBuffer {
		logChan = vv104.NewLogChan()
		go readLogChan()
	}

	go state.Start()

}

func disconnectIec104() {
	state.Chans.CommandsFromStdin <- "disconnect"
	cancel()
}

func readLogChan() {
	fmt.Println("readLogChan start")

	// it is maybe not necessary anymore for this to be so complicated
	// we could just log to a string directly.
	// however there was a race condition that is hopefully now resolved with the waitgroup, and this function was one attempt to fix it

	for {
		select {
		case s := <-logChan:
			wg.Wait()
			logTxt += s
			giu.Update()

		case <-ctx.Done():
			fmt.Println("Done, readLogChan exits")
			return
		}

	}
}

func addObject() {

	asdu := vv104.NewAsdu()
	typeIdStr = typeIds[typeIdSelected]
	asdu.TypeId = vv104.TypeIdFromName(typeIdStr)

	var infoObj vv104.InfoObj
	infoObj.Ioa = vv104.Ioa(ioa)
	asdu.AddInfoObject(infoObj)
	err := state.Objects.AddObjectByName(objName, *asdu)
	if err != nil {
		fmt.Println(err)
	}
	// ioa++

}

func sendMoniObject() {
	objName := strings.Split(state.Objects.MoniList[moniObjectSelected], " ")[0]

	asdu, ok := state.Objects.MoniObjects[objName]

	if ok {
		sendObject(asdu)
	} else {
		fmt.Println("can't send, object not found")
	}
}

func sendCtrlObject() {
	objName := strings.Split(state.Objects.CtrlList[ctrlObjectSelected], " ")[0]

	asdu, ok := state.Objects.CtrlObjects[objName]

	if ok {
		sendObject(asdu)
	} else {
		fmt.Println("can't send, object not found")
	}
}

func sendObject(asdu vv104.Asdu) {

	if state.TcpConnected {

		apdu := vv104.NewApdu()

		err := checkValue(asdu.TypeId == vv104.M_ME_NC_1 || asdu.TypeId == vv104.M_ME_TF_1)

		if err != nil {
			fmt.Println(err)
			return
		}

		if asdu.TypeId == vv104.M_ME_NC_1 || asdu.TypeId == vv104.M_ME_TF_1 {
			if _, ok := value.(vv104.FloatVal); !ok {
				fmt.Println("Value must be float")
				return
			}
		} else if _, ok := value.(vv104.IntVal); !ok {
			fmt.Println("Value must be Int")
			return
		}
		asdu.InfoObj[0].Value = value
		asdu.InfoObj[0].TimeTag = time.Now()
		// asdu.InfoObj[0].
		asdu.Casdu = vv104.Casdu(casdu)
		asdu.CauseTx = vv104.CauseTxFromName(causeTxs[causeTxSelected])
		apdu.Asdu = asdu
		apdu.Apci.FrameFormat = vv104.IFormatFrame

		state.Chans.ToSend <- apdu
	} else {
		fmt.Println("can't send, no connection")
	}

}

func removeMoniObject() {
	removeObject(moniObjectSelected, int32(len(state.Objects.MoniList)), state.Objects.MoniList)
}
func removeCtrlObject() {
	removeObject(ctrlObjectSelected, int32(len(state.Objects.CtrlList)), state.Objects.CtrlList)
}

func removeObject(objectSelected int32, lenList int32, list vv104.ObjectList) {

	if objectSelected < lenList && objectSelected >= 0 && lenList >= 1 {
		// ObjectList contains objName | TypeID | IOA, we only need objName for reference, so we have to cut after first space
		// this is a bit of a hack. Todo.
		objName := strings.Split(list[objectSelected], " ")[0]
		err := state.Objects.RemoveObject(objName)
		if err != nil {
			fmt.Println(err)
		}

	}

}

func checkIpAddr() {
	ip := net.ParseIP(ipAddrStr)

	if ip == nil {
		errorTextIp = "<- Enter valid IP Address"

	} else {
		errorTextIp = ""
	}
}

func checkPort() {

	port, err := strconv.Atoi(portStr)
	if (port < 0) || (port > 65535) || err != nil {
		errorTextPort = ("<- Enter a Port between 0 and 65535")
	} else {
		errorTextPort = ""
	}
}

func checkCasdu() {
	var err error
	casduInt, err := strconv.Atoi(casduStr)
	if casduInt > 65535 || casduInt < 1 || err != nil {
		errorTextCasdu = "<- Enter valid Casdu"
	} else {
		errorTextCasdu = ""
		casdu = int16(casduInt)
	}
}

func checkValue(checkForFloat bool) error {

	var err error
	var valueFloat64 float64

	if checkForFloat {

		valueFloat64, err = strconv.ParseFloat(valueStr, 32)
		if err != nil {
			return errors.New("cant convert value to float")
		}
		value = vv104.FloatVal(float32(valueFloat64))

	} else {
		// check for int
		var valueInt int
		valueInt, err = strconv.Atoi(valueStr)

		if err != nil {
			return errors.New("cant convert value to int")
		}

		if valueInt > 32767 || valueInt < -32768 {
			// todo
			return errors.New("value out of range")
		}

		// is valid int value
		value = vv104.IntVal(valueInt)

	}

	fmt.Println("value:", value)
	return nil
}

func openConfig() {
	filename, err := dialog.File().Filter("TOML file", "toml").Load()

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(filename)

	loadedConfig, loadedObjects, err := vv104.LoadConfigAndObjectsFromFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	state.Config = *loadedConfig
	state.Objects = loadedObjects

	if state.TcpConnected {
		disconnectIec104()
	}

	// copy values from config to GUI, TODO, add new fields
	ipAddrStr = state.Config.Ipv4Addr
	portStr = fmt.Sprint(state.Config.Port)
	mode = state.Config.Mode
	k = fmt.Sprint(state.Config.K)
	w = fmt.Sprint(state.Config.W)
	t1 = fmt.Sprint(state.Config.T1)
	t2 = fmt.Sprint(state.Config.T2)
	t3 = fmt.Sprint(state.Config.T1)
}

func saveConfig() {
	writeConfigsToState()
	fileName, err := dialog.File().Filter("TOML file", "toml").Title("Save config as .toml file").Save()
	if err != nil {
		fmt.Println(err)
	}
	err = vv104.WriteConfigAndObjectsToFile(&state, fileName)
	if err != nil {
		fmt.Println(err)
	}
}

func writeConfigsToState() {
	state.Config.Mode = mode
	state.Config.Ipv4Addr = ipAddrStr
	state.Config.Port, _ = strconv.Atoi(portStr)

	state.Config.K, _ = strconv.Atoi(k)
	state.Config.W, _ = strconv.Atoi(w)
	state.Config.T1, _ = strconv.Atoi(t1)
	state.Config.T2, _ = strconv.Atoi(t2)
	state.Config.T3, _ = strconv.Atoi(t3)
	state.Config.AutoAck = true
	state.Config.IoaStructured = false
	state.Config.UseLocalTime = false

	state.Config.InteractiveMode = true
	state.Config.LogToBuffer = true
	state.Config.LogToStdOut = true

}

func sendGI() {
	gi := vv104.NewApdu()
	infoObj := vv104.NewInfoObj()
	infoObj.Ioa = 0
	infoObj.CommandInfo.Qoi = 20
	gi.Apci.FrameFormat = vv104.IFormatFrame
	gi.Asdu.TypeId = vv104.C_IC_NA_1
	gi.Asdu.CauseTx = vv104.Act
	gi.Asdu.Casdu = vv104.Casdu(casdu)
	gi.Asdu.AddInfoObject(infoObj)

	state.Chans.ToSend <- gi
}
