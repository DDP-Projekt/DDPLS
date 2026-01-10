# DDPLS
Der Sprach-Server [der Deutschen Programmiersprache](https://github.com/DDP-Projekt/Kompilierer). 
Mit einem Sprach-Server ist es möglich IDE's zu erweitern, um DDP zu unterstützen.

## Funktionen
<!-- TOC -->
- [DDPLS](#ddpls)
	- [Funktionen](#funktionen)
		- [Syntaxhervorhebung](#Syntaxhervorhebung)
		- [Diagnose Informationen](#diagnose-informationen)
		- [Hover](#hover)
			- [Über einer Variable](#über-einer-variable)
			- [Über einem Funktionsaufruf](#über-einem-funktionsaufruf)
		- [Go to definition und Peek definition](#go-to-definition-und-peek-definition)
		- [Vorschläge](#vorschläge)
		- [Umbennenen](#umbennenen)
		- [Variablen Hervorhebung](#variablen-hervorhebung)
<!-- TOC -->

### Syntaxhervorhebung
Der Sprach-Server vergibt je nach Kontext, bestimmten Bereichen des Quellcodes Attribute<br>
So können erweiterungen die Attribute des Sprach-Servers benutzen, um diesen Textabschnitten verschiedene Farben zu vergeben.
![highlighting img](https://i.imgur.com/DZIJ4pd.png)

### Diagnose Informationen
Der Sprach-Server schickt Fehlermeldungen des Kompilierers an die Erweiterung, sodass sie dem Nutzer angezeigt werden kann.

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
Während dem Tippen kann der Sprach-Server dir Vorschläge für Schlüsselwörter, Typen, Funktionen und Variablen geben.
![completion img](https://i.imgur.com/HbjB9pQ.png)

### Umbennenen
Man kann Variablen, Funktionsparameter und Kombinationsparameter umbennen.
![rename](https://i.imgur.com/ezaTUBv.png)

### Variablen Hervorhebung
Variablen und Parameter werden hervorgehoben wenn die Maus neben ihr liegt
![highlight](https://i.imgur.com/h5rs2ye.png)
