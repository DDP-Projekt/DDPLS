# DDPLS
Der Language Server [der Deutschen Programmiersprache](https://github.com/DDP-Projekt/Kompilierer). 
Mit einem Language Server ist es möglich IDE's zu erweitern, um DDP zu unterstützen.

## Features
<!-- TOC -->
* [Syntax highlighting](#syntax-highlighting)
* [Diagnose Informationen](#diagnose-informationen)
* [Hover](#hover)
    * [Über einer Variable:](#ber-einer-variable)
    * [Über einem Funktionsaufruf:](#ber-einem-funktionsaufruf)
* [Go to definition und Peek definition](#go-to-definition-und-peek-definition)
* [Vorschläge](#vorschlge)
<!-- TOC -->

### Syntax highlighting
Der Language server vergibt je nach Kontext, bestimmten Bereichen des Quellcodes Attribute<br>
So können erweiterungen die Attribute des Language Servers benutzen, um diesen Textabschnitten verschiedene Farben zu vergeben.
![highlighting img](https://i.imgur.com/DZIJ4pd.png)

### Diagnose Informationen
Der Language server schickt Fehlermeldungen des Kompilierers an die Erweiterung, sodass sie dem Nutzer angezeigt werden kann.

### Hover
Zeigt die Variablen-/Funktionsdeklaration und ihrer Position, wenn die Maus über eine Variable oder einen Funktionsaufruf schwebt.

#### Über einer Variable
![hover var img](https://i.imgur.com/334ijIb.png)

#### Über einem Funktionsaufruf
![hover func img](https://i.imgur.com/mgpeuvu.png)

### Go to definition und Peek definition
Mit "Go to definition" kann man schnell zu einer Variablen- oder Funktionsdeklaration springen.

Peek definition erlaubt dem Nutzer schnell eine Variablen- oder Funktionsdeklaration einzusehen, wie hier im Bild:
![go to def img](https://i.imgur.com/9edLyuO.png)

### Vorschläge
Während dem Tippen kann der Language Server dir Vorschläge für Schlüsselwörter, Typen, Funktionen und Variablen geben.
![completion img](https://i.imgur.com/HbjB9pQ.png)