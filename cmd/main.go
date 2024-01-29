package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/viduq/vv104"

	"github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"github.com/sqweek/dialog"
	"golang.org/x/image/colornames"
)

var (
	sashPos1           float32  = 100
	sashPos2           float32  = 250
	colorConnection             = colornames.Cadetblue
	typeIds            []string = vv104.TypeIDs
	typeIdSelected     int32
	causeTxs           []string = vv104.CauseTxs
	causeTxSelected    int32
	typeIdStr          string
	valueStr           string = "0"
	valueInt           int    = 0
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

	state   vv104.State
	objects vv104.Objects

	ctx     context.Context
	cancel  context.CancelFunc
	logChan <-chan string
	lines   []*giu.TableRowWidget
)

func init() {
	objects = *vv104.NewObjects()
}

func loop() {

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
								giu.ListBox("Monitoring", objects.MoniList).SelectedIndex(&moniObjectSelected).Size(sashPos2, 300),
								giu.Button("Remove Moni Obj.").OnClick(removeMoniObject),
								giu.Button("Send M. Obj.").OnClick(sendMoniObject).Size(100, 30),
							), //.IsOpen(&moniObjectsTabIsOpen), //.Flags(giu.TabItemFlagsSetSelected),
							giu.TabItem("Control").Layout(
								giu.ListBox("Control", objects.CtrlList).SelectedIndex(&ctrlObjectSelected).Size(sashPos2, 300),
								giu.Button("Remove Ctrl Obj.").OnClick(removeCtrlObject),
								giu.Button("Send C. Obj.").OnClick(sendCtrlObject).Size(100, 30),
							), //.IsOpen(&ctrlObjectsTabIsOpen),
						),
						giu.Combo("Cause Tx", causeTxs[causeTxSelected], causeTxs, &causeTxSelected),
						giu.InputText(&valueStr).OnChange(checkValue),
					),
				},
				giu.Layout{
					giu.Row(
						giu.Checkbox("Auto Scrolling", &autoScrolling),
					),
					// giu.Label("Log"),
					// giu.InputTextMultiline(&logTxt).AutoScrollToBottom(autoScrolling).Size(1000, 500).Flags(giu.InputTextFlagsReadOnly),
					// giu.ListBox("LogBox", logTxt).SelectedIndex(&selectedIndex),
					giu.Table().Columns(giu.TableColumn("")).Rows(lines...),
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

	for {
		select {
		case s, ok := <-logChan:
			if !ok {
				// channel is closed
				fmt.Println("readLogChan returns")
				return
			}

			lines = append(lines, giu.TableRow(giu.Custom(func() {
				giu.Labelf(s).Build()

				if autoScrolling {
					imgui.SetScrollHereY(1.0)
				}
				giu.Update()

			})))
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
	err := objects.AddObjectByName(objName, *asdu)
	if err != nil {
		fmt.Println(err)
	}
	// ioa++

}

func sendMoniObject() {
	objName := strings.Split(objects.MoniList[moniObjectSelected], " ")[0]

	asdu, ok := objects.MoniObjects[objName]

	if ok {
		sendObject(asdu)
	} else {
		fmt.Println("can't send, object not found")
	}
}

func sendCtrlObject() {
	objName := strings.Split(objects.CtrlList[ctrlObjectSelected], " ")[0]

	asdu, ok := objects.CtrlObjects[objName]

	if ok {
		sendObject(asdu)
	} else {
		fmt.Println("can't send, object not found")
	}
}

func sendObject(asdu vv104.Asdu) {

	if state.TcpConnected {

		apdu := vv104.NewApdu()

		asdu.InfoObj[0].Value = vv104.IntVal(valueInt)
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
	removeObject(moniObjectSelected, int32(len(objects.MoniList)), objects.MoniList)
}
func removeCtrlObject() {
	removeObject(ctrlObjectSelected, int32(len(objects.CtrlList)), objects.CtrlList)
}

func removeObject(objectSelected int32, lenList int32, list vv104.ObjectList) {
	// fmt.Println(moniObjectSelected)
	// fmt.Println(ctrlObjectSelected)

	// var objectSelected int32
	// var lenList int32
	// var list vv104.ObjectList

	// // This does not work yet, moniObjectsTabIsOpen is not
	// if moniObjectsTabIsOpen {
	// 	objectSelected = moniObjectSelected
	// 	lenList = int32(len(objects.MoniList))
	// 	list = objects.MoniList
	// } else {
	// 	objectSelected = ctrlObjectSelected
	// 	lenList = int32(len(objects.CtrlList))
	// 	list = objects.CtrlList
	// }

	if objectSelected < lenList && objectSelected >= 0 && lenList >= 1 {
		// ObjectList contains objName | TypeID | IOA, we only need objName for reference, so we have to cut after first space
		// this is a bit of a hack. Todo.
		objName := strings.Split(list[objectSelected], " ")[0]
		err := objects.RemoveObject(objName)
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

func checkValue() {

	var err error
	valueInt, err = strconv.Atoi(valueStr)

	if err != nil {
		fmt.Println("cant convert value to int")
	}

	if valueInt > 32767 || valueInt < -32768 {
		// todo
		fmt.Println("value out of range")

	}

	fmt.Println("value:", valueInt)

	// to do: check if value is float for float types.
	// or is int for all other types
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
	objects = *loadedObjects

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
	err = vv104.WriteConfigAndObjectsToFile(state.Config, objects, fileName)
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
