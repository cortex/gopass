import QtQuick 2.5
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
  minimumWidth: mainLayout.Layout.minimumWidth + 2 * margin
  minimumHeight: 400
  x: (Screen.width - width) / 2
  flags: Qt.FramelessWindowHint | Qt.Window
  color: "transparent"

  MouseArea {
    id: mouseRegion
    property variant clickPos: "1,1"

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

        Layout.minimumWidth: 100
        Layout.fillHeight: true
        Layout.fillWidth: true

        TextField {
          id: searchInput

          height: 42
          Layout.fillWidth: true
          font.pixelSize: 24
          focus: true

          onTextChanged: passwords.query(text)
          onAccepted: passwords.copyToClipboard(hitList.currentIndex)
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
          }
        }

        Text {
          id: status

          Layout.fillWidth: true
          text: passwords.status
          z: -1
          height: 14
          font.pixelSize: 14
          color: "#aaa"
        }
      }

      ColumnLayout {
        id: rightPane

        Layout.maximumWidth: 300
        Layout.minimumWidth: 200
        Layout.fillHeight: true
        Layout.fillWidth: true

        Rectangle {
          id: frame

          Layout.fillWidth: true
          Layout.fillHeight: true
          color: "#444"
          border.color: "#333"
          border.width: 2
          radius: 10

          Image {
            id: logo

            width: 48
            height: 48
            anchors.top: parent.top
            anchors.topMargin: 24
            anchors.horizontalCenter: parent.horizontalCenter
            fillMode: Image.PreserveAspectFit
            source: "logo.svg"
          }
          Canvas{
            id: progress
            property double countdown: passwords.countdown

            width: 100; height: 100
            anchors.top: parent.top
            anchors.horizontalCenter: parent.horizontalCenter
            contextType: "2d"

            onCountdownChanged: progress.requestPaint()
            onPaint: {
              var top = 3.0*(Math.PI/2.0)
              var cx = 50, cy = 50, r = 40, lw = 5
              var p = (countdown/15.0)
              context.reset()
              context.lineWidth = lw
              context.strokeStyle = "#666"
              context.arc(cx, cy, r, 0, 2.0*Math.PI , false)
              context.stroke()

              context.beginPath()
              context.strokeStyle = "#a6a"
              context.lineWidth = lw
              context.arc(cx, cy, r, top, top-p*2.0*Math.PI, false)
              context.stroke()
            }

          }

          Item {
            id: metadataContainer

            anchors.top: logo.bottom
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            anchors.margins: 8
            anchors.topMargin: 24

            TextEdit {
              id: metadata
              anchors.fill: parent
              selectByMouse: true
              readOnly: true
              font.pixelSize: 12
              font.family: "Courier"
              color: "#999"
              selectionColor: "#666"
              text: passwords.metadata
              wrapMode: TextEdit.WordWrap
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
      onActivated: passwords.clearmetadata()
    }

    Shortcut {
      sequence:"Ctrl+l"
      onActivated: {searchInput.selectAll(); searchInput.focus=true}
    }

    Shortcut {
      sequence: "Esc"
      context: Qt.ApplicationShortcut
      onActivated: passwords.quit()
    }

    Component {
      id: passwordEntry

      Text {
        property var view: ListView.view
        property int itemIndex: index

        text: passwords.password(index).name;
        font.pixelSize: 18
        color: ListView.isCurrentItem? "#dd00bb":"gray"

        MouseArea{
          anchors.fill: parent
          onClicked: view.currentIndex = itemIndex
        }
      }
    }
  }
}
