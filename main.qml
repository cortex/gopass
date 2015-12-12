import QtQuick 2.5
import QtQuick.Controls 1.3
import QtQuick.Layouts 1.2
import QtQuick.Controls.Styles 1.4

ApplicationWindow {
    id: rootWindow
    visible: true
    title: "GoPass"
    property int margin: 10
    width: 500
    height: mainLayout.implicitHeight + 2 * margin
    minimumWidth: mainLayout.Layout.minimumWidth + 2 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 2 * margin

    flags: Qt.FramelessWindowHint | Qt.Window
    color: "transparent"

    Rectangle {
        color: "#333"
        anchors.fill: parent
        anchors.margins: 0
        radius: 10
        border.width: 3
        border.color: "white"
    }

    MouseArea {
        id: mouseRegion
        anchors.fill: parent;
        property variant clickPos: "1,1"

        onPressed: {
            clickPos  = Qt.point(mouse.x,mouse.y)
        }

        onPositionChanged: {
            var delta = Qt.point(mouse.x-clickPos.x, mouse.y-clickPos.y)
            rootWindow.x += delta.x;
            rootWindow.y += delta.y;
        }
    }
    Shortcut {
        id: up
        sequence:"Ctrl+K"
        context: Qt.ApplicationShortcut
        onActivated: passwords.up()
    }

    Shortcut {
        id: down
        sequence:"Ctrl+j"
        onActivated: passwords.down()
    }

    Shortcut {
        id: selectAll
        sequence:"Ctrl+l"
        onActivated: searchInput.selectAll()
    }
    Component {
        id: passwordEntry
        Text {
            text: passwords.password(index).name;
            font.pixelSize: 24
            color: index==passwords.selected ? "#dd00bb": "gray"
        }
    }

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin

        TextField {
            id: searchInput
            font.pixelSize: 24
            focus: true
            Layout.fillWidth: true
            onTextChanged: passwords.query(text)
            onAccepted: passwords.copy()
            textColor: "white"
            style: TextFieldStyle {
                textColor: "white"
                background: Rectangle {
                    radius: 5
                    border.color: "#666"
                    border.width: 1
                    color: "#333"
                }
            }
        }
        ListView {
            id: hitList
            model: passwords.len
            delegate: passwordEntry
            Layout.fillHeight: true
            Layout.minimumHeight: 300
        }
    }
}