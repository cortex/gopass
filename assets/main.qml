import QtQuick 2.5
import QtQuick.Controls 1.3
import QtQuick.Layouts 1.2
import QtQuick.Controls.Styles 1.4
import QtQuick.Window 2.2

ApplicationWindow {
  id: rootWindow
  visible: true
  title: "GoPass"
  property int margin: 10
  width: 600
  height: mainLayout.implicitHeight + 2 * margin
  minimumWidth: mainLayout.Layout.minimumWidth + 2 * margin
  minimumHeight: 400

  x: (Screen.width - width) / 2

  flags: Qt.FramelessWindowHint | Qt.Window
  color: "transparent"

  Rectangle {
    color: "#333"
    anchors.fill: parent
    anchors.margins: 0
    radius: 10
    border.width: 2
    border.color: "#aaa"
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
    sequence:"Ctrl+K"
    context: Qt.ApplicationShortcut
    onActivated: passwords.up()
  }

  Shortcut {
    sequence:"Up"
    context: Qt.ApplicationShortcut
    onActivated: passwords.up()
  }

  Shortcut {
    sequence:"Ctrl+j"
    onActivated: passwords.down()
  }

  Shortcut {
    sequence:"Down"
    onActivated: passwords.down()
  }

  Shortcut {
    sequence:"Ctrl+r"
    onActivated: passwords.clearmetadata()
  }

  Shortcut {
    sequence:"Ctrl+l"
    onActivated: searchInput.selectAll()
  }

  Shortcut {
    sequence: "Esc"
    context: Qt.ApplicationShortcut
    onActivated: passwords.quit()
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
    RowLayout{
      Layout.fillWidth: true
      TextField {
        id: searchInput
        font.pixelSize: 24
        focus: true
        onTextChanged: passwords.query(text)
        onAccepted: passwords.copyToClipboard()
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
        Layout.fillWidth: true
      }
      Image {
        source: "logo.svg"
        fillMode: Image.PreserveAspectFit
        Layout.alignment: Qt.AlignRight
        Layout.maximumWidth: 32
      }
    }
    RowLayout{
      ColumnLayout{
        ScrollView{
          Layout.fillHeight: true
          Layout.fillWidth: true
          style: ScrollViewStyle{
            transientScrollBars: true
            scrollToClickedPosition : true
          }
          ListView {
            id: hitList
            model: passwords.len
            delegate: passwordEntry
            Layout.fillHeight: true
          }
        }
        Text {
          id: status
          text: passwords.status
          height: 14
          font.pixelSize: 14
          color: "#aaa"
        }
      }
      ScrollView{
        Layout.fillHeight: true
        Layout.preferredWidth: 250 
        style: ScrollViewStyle{
          transientScrollBars: true
          scrollToClickedPosition : true
        }
        Rectangle {
          anchors.fill: parent
          color: "#444"
          border.color: "#333"
          border.width: 2
          radius: 10
        }
        TextEdit {
          id: metadata
          selectByMouse: true
          readOnly: true
          text: passwords.metadata
          font.pixelSize: 14
          color: "#999"
          wrapMode: TextEdit.WordWrap
        }
      }
    }
  }
}
