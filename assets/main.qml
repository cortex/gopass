import QtQuick 2.7
import QtQuick.Controls 1.3
import QtQuick.Layouts 1.2
import QtQuick.Controls.Styles 1.4
import QtQuick.Window 2.2

ApplicationWindow {
    id: rootWindow

    property int margin: 10

    visible: true
    title: "GoPass"
    width: 800
    height: 400
    minimumHeight: 400
    x: (Screen.width - width) / 2
    flags: Qt.FramelessWindowHint | Qt.Window
    color: "transparent"

    MouseArea {
        id: mouseRegion
        property var clickPos: "1, 1"
        anchors.fill: parent;

        onPressed: {
            clickPos  = Qt.point(mouse.x,mouse.y)
        }

        onPositionChanged: {
            var delta = Qt.point(mouse.x-clickPos.x, mouse.y-clickPos.y)
            rootWindow.x += delta.x;
            rootWindow.y += delta.y;
        }
    }

    Rectangle {
        id: mainLayout

        color: "#333"
        radius: 10
        anchors.fill: parent
        border.width: 2
        border.color: "#aaa"

        RowLayout {
            id: panes

            anchors.fill: parent
            anchors.margins: 8

            ColumnLayout {
                id: leftPane
                Layout.fillHeight: true
                Layout.fillWidth: true

                TextField {
                    id: searchInput

                    height: 42
                    Layout.fillWidth: true
                    font.pixelSize: 24
                    focus: true

                    onTextChanged: ui.query(text)
                    onAccepted: passwords.copyToClipboard(hitList.currentIndex)

                    placeholderText: "Search your passwords..."

                    style: TextFieldStyle {
                        textColor: "white"
                        placeholderTextColor: "#444"
                        background: Rectangle {
                            radius: 5
                            border.color: "#666"
                            border.width: 1
                            color: "#333"
                        }
                    }
                }

                ScrollView{
                    id: resultsContainer

                    Layout.fillHeight: true
                    Layout.fillWidth: true
                    style: ScrollViewStyle{
                        transientScrollBars: true
                        scrollToClickedPosition : true
                    }

                    ListView {
                        id: hitList

                        anchors.fill: parent
                        focus: true
                        interactive: true
                        model: passwords.len
                        delegate: passwordEntry
                        highlight: Rectangle {
                            color: "#444"
                            radius: 3
                            anchors.left: parent ? parent.left : undefined
                            anchors.right: parent ? parent.right : undefined
                        }
                        onCurrentItemChanged:{
                            passwords.select(currentIndex)
                        }
                    }
                }

                Text {
                    id: status
                    Layout.fillWidth: true
                    text: ui.status
                    z: -1
                    height: 14
                    font.pixelSize: 14
                    color: "#aaa"
                }
            }

            Rectangle {
                id: frame
                width: 300;
                Layout.fillHeight: true
                color: "#444"
                radius: 10

                ColumnLayout {
                    id: rightPane
                    anchors.fill: parent
                    spacing: 2
                    Item {
                        id: logoBox

                        Layout.minimumHeight: 100
                        Layout.maximumHeight: 100
                        Layout.fillWidth: true
                        MouseArea{
                            onClicked: ui.toggleShowMetadata()
                            width: 48; height: 48
                            anchors.left: parent.left
                            anchors.leftMargin: 25
                            anchors.verticalCenter: parent.verticalCenter
                        Image{
                            id: metadataToggle
                            width: 48; height: 48
                            fillMode: Image.PreserveAspectFit
                            source: ui.showMetadata ? "eye_open.svg": "eye_closed.svg"

                        }
                        }
                        Image {
                            id: logo
                            width: 48
                            height: 48
                            anchors.verticalCenter: parent.verticalCenter
                            anchors.horizontalCenter: parent.horizontalCenter
                            anchors.topMargin: 24
                            fillMode: Image.PreserveAspectFit
                            source: "logo.svg"
                        }

                        Image{
                            id: copyIcon
                            width: 48; height: 48
                            anchors.right: parent.right
                            anchors.rightMargin: 25
                            fillMode: Image.PreserveAspectFit
                            source: "copy.svg"
                            anchors.verticalCenter: parent.verticalCenter
                            MouseArea{
                                anchors.fill: parent
                                onClicked: passwords.copyToClipboard(hitList.currentIndex)
                            }
                        }

                        Canvas{
                            id: progress
                            property double countdown: ui.countdown
                            property bool cached: ui.password.cached

                            width: 100; height: 100
                            anchors.verticalCenter: parent.verticalCenter
                            anchors.horizontalCenter: parent.horizontalCenter
                            visible: true
                            contextType: "2d"

                            onCountdownChanged: progress.requestPaint()
                            onPaint: {
                                var top = 3.0*(Math.PI/2.0)
                                var cx = 50, cy = 50, r = 40, lw = 5
                                var p = (countdown/15.0)
                                context.reset()
                                context.lineWidth = lw
                                context.strokeStyle = cached?"#666":"#966"
                                context.arc(cx, cy, r, 0, 2.0*Math.PI , false)
                                context.stroke()

                                context.beginPath()
                                context.strokeStyle = "#a6a"
                                context.lineWidth = lw
                                context.arc(cx, cy, r, top, top-p*2.0*Math.PI, false)
                                context.stroke()
                            }
                        }
                    }
                    Text{
                        Layout.fillWidth: true;
                        anchors.horizontalCenter: parent.horizontalCenter
                        horizontalAlignment: Text.AlignHCenter
                        font.pixelSize: 18
                        text: ui.password.name
                        color: "#eee"
                    }
                    Rectangle {
                        id: rectangle1
                        height: 24
                        color: "#555"
                        border.color: "#444"
                        border.width: 2
                        radius: 10
                        Layout.fillHeight: false
                        Layout.fillWidth: true
                        Layout.maximumHeight: 24
                        Layout.minimumHeight: 24
                        Layout.margins: 5
                        Text{
                            id: info
                            horizontalAlignment: Text.AlignHCenter
                            color: "#aaa"
                            padding: 5
                            font.pixelSize: 10
                            text: ui.password.info
                            anchors.horizontalCenter: parent.horizontalCenter
                       }
                    }

                    /*
                    RowLayout {
                        id: rowLayout1
                        width: 100
                        height: 100
                        Layout.fillWidth: true
                        anchors.horizontalCenter: parent.horizontalCenter

                        RoundButton {
                            label: "COPY"
                            onClicked: passwords.copyToClipboard(hitList.currentIndex)
                        }
                        RoundButton {
                            label: "SHOW"
                            onClicked: ui.toggleShowMetadata()
                        }

                    }
*/
                    ScrollView {
                        id: metadataContainer
                        Layout.fillHeight: true
                        Layout.fillWidth: true
                        Layout.margins: 10
                        style: ScrollViewStyle{
                            transientScrollBars: true
                            scrollToClickedPosition : true
                        }
                        TextEdit {
                            id: metadata
                            width: 270
//                            anchors.fill: metadataContainer
                            selectByMouse: true
                            readOnly: true
                            font.pixelSize: 12
                            font.family: "Courier"
                            color: "white"
                            selectionColor: "#666"
                            text: ui.password.metadata
                            wrapMode: TextEdit.WrapAnywhere
                        }
                    }
                }
            }
        }


        Shortcut {
            sequence:"Ctrl+K"
            context: Qt.ApplicationShortcut
            onActivated: hitList.decrementCurrentIndex()
        }

        Shortcut {
            sequence:"Up"
            context: Qt.ApplicationShortcut
            onActivated: hitList.decrementCurrentIndex()
        }

        Shortcut {
            sequence:"Ctrl+j"
            onActivated: hitList.incrementCurrentIndex()
        }

        Shortcut {
            sequence:"Down"
            onActivated: hitList.incrementCurrentIndex()
        }

        Shortcut {
            sequence:"Ctrl+r"
            onActivated: ui.toggleShowMetadata()
        }

        Shortcut {
            sequence:"Ctrl+l"
            onActivated: {searchInput.selectAll(); searchInput.focus=true}
        }

        Shortcut {
            sequence: "Esc"
            context: Qt.ApplicationShortcut
            onActivated: ui.quit()
        }

        Component {
            id: passwordEntry

            Text {
                property var view: ListView.view
                property int itemIndex: index

                text: passwords.get(index);
                font.pixelSize: 18
                color: ListView.isCurrentItem? "#dd00bb":"gray"

                MouseArea{
                    anchors.fill: parent
                    onClicked: view.currentIndex = itemIndex
                    onDoubleClicked: {
                        clicked(passwordEntry)
                        passwords.copyToClipboard(hitList.currentIndex)
                    }
                }
            }
        }
    }
}
