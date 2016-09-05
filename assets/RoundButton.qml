import QtQuick 2.3

Rectangle {
    id: container
    width: 100; height: 25
    border.color: btnColor
    color:"transparent"
    radius: 10
    property string label: "label"
    property var btnColor: "#999"
    signal clicked()
    MouseArea {
        anchors.fill: parent
        onClicked: container.clicked()
        hoverEnabled: true
        onEntered: {
           parent.color="#666" 
        }
        onExited: {
           parent.color="transparent" 
        }
    }
    Text {
        anchors.fill: parent
        anchors.horizontalCenter: parent.horizontalCenter
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
        text:parent.label
        color:parent.btnColor
        font.pixelSize: 10
    }
}