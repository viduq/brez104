package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/viduq/vv104"

	"github.com/AllenDang/giu"
	"golang.org/x/image/colornames"
)

var (
	sashPos1             float32 = 100
	sashPos2             float32 = 250
	logTxt               string
	colorConnection               = colornames.Cadetblue
	typeIds              []string = vv104.TypeIDs
	typeIdSelected       int32
	typeIdStr            string
	moniObjectSelected   int32
	ctrlObjectSelected   int32
	moniObjectsTabIsOpen bool             = true
	ctrlObjectsTabIsOpen bool             = false
	tabItemSelected      giu.TabItemFlags = giu.TabItemFlagsSetSelected

	ioa           int32 = 1
	objName       string
	ipAddrStr     string = "127.0.0.1"
	portStr       string = "2404"
	mode          string = "client"
	autoScrolling bool   = true
	errorTextIp   string = ""
	errorTextPort string = ""
	k             string = "12"
	w             string = "8"
	t1            string = "15"
	t2            string = "10"
	t3            string = "20"

	state   vv104.State
	objects vv104.Objects
)

func init() {
	objects = *vv104.NewObjects()
}

func loop() {

	giu.SingleWindowWithMenuBar().Layout(
		giu.MenuBar().Layout(
			giu.Menu("File").Layout(
				giu.MenuItem("Open Config").Shortcut("Ctrl+O"),
			),
		),
		giu.SplitLayout(giu.DirectionHorizontal, &sashPos1,
			giu.Layout{
				giu.Label("Configuration"),
				giu.Column(
					giu.Row(
						giu.RadioButton("Client", mode == "client").OnChange(func() { mode = "client"; fmt.Println("client") }),
						giu.RadioButton("Server", mode == "server").OnChange(func() { mode = "server"; fmt.Println("server") }),
						giu.Label("IP:"),
						giu.InputText(&ipAddrStr).OnChange(checkIpAddr).Label("###ipInput").Size(150),
						giu.Label(errorTextIp),
						giu.Label("Port:"),
						giu.InputText(&portStr).Label("###portInput").Size(50).Flags(giu.InputTextFlagsCharsDecimal).OnChange(checkPort),
						giu.Label(errorTextPort),

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
								giu.Button("Remove Moni Obj.").OnClick(removeMoniObject),
								giu.ListBox("Monitoring", objects.MoniList).SelectedIndex(&moniObjectSelected).Size(sashPos2, 300),
							), //.IsOpen(&moniObjectsTabIsOpen), //.Flags(giu.TabItemFlagsSetSelected),
							giu.TabItem("Control").Layout(
								giu.Button("Remove Ctrl Obj.").OnClick(removeCtrlObject),

								giu.ListBox("Control", objects.CtrlList).SelectedIndex(&ctrlObjectSelected).Size(sashPos2, 300),
							), //.IsOpen(&ctrlObjectsTabIsOpen),
						),
					),
				},
				giu.Layout{
					giu.Row(
						giu.Checkbox("Auto Scrolling", &autoScrolling).OnChange(func() { fmt.Println(autoScrolling) }),
					),
					// giu.Label("Log"),
					giu.InputTextMultiline(&logTxt).AutoScrollToBottom(autoScrolling).Size(1000, 1000).Flags(giu.InputTextFlagsReadOnly),
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

	state.Config.Mode = mode
	state.Config.Ipv4Addr = ipAddrStr
	state.Config.Port, _ = strconv.Atoi(portStr)
	state.Config.InteractiveMode = true
	state.Config.LogToBuffer = true
	state.Config.LogToStdOut = true

	state.Config.K, _ = strconv.Atoi(k)
	state.Config.W, _ = strconv.Atoi(w)
	state.Config.T1, _ = strconv.Atoi(t1)
	state.Config.T2, _ = strconv.Atoi(t2)
	state.Config.T3, _ = strconv.Atoi(t3)
	state.Config.AutoAck = true
	state.Config.IoaStructured = false
	state.Config.UseLocalTime = false

	vv104.LogCallBack = logCallback
	// go refresh()
	go state.Start()

}

func disconnectIec104() {
	state.Chans.CommandsFromStdin <- "disconnect"
}

func logCallback() {
	logTxt += vv104.ReadLogEntry()
	giu.Update()
}

func addObject() {
	fmt.Println(moniObjectsTabIsOpen)
	fmt.Println(ctrlObjectsTabIsOpen)

	asdu := vv104.NewAsdu()
	typeIdStr = typeIds[typeIdSelected]
	asdu.TypeId = vv104.TypeIdFromName(typeIdStr)

	var infoObj vv104.InfoObj
	infoObj.Ioa = vv104.Ioa(ioa)
	asdu.InfoObj = append(asdu.InfoObj, infoObj)
	err := objects.AddObject(objName, *asdu)
	if err != nil {
		fmt.Println(err)
	}
	// ioa++

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
