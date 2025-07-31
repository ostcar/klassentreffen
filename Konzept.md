# Klassentrefen

Es geht um eine Webanwendung, um ein Klassentreffen zu planen. Ziel ist, dass
alle Teilnehmer sich anmelden können und eine Liste aller teilnehmer mit den
Kontaktdaten entstehe.


## Worflow

Beim Besutz der Webseite sieht ein Nutzer nur ein Feld zur Eingabe einer
E-Mail-Adresse und einen "Anmelde" Button. Beim klick auf diesen Button wird ein
POST-Request an den Server gesendet. Dieser sendet zur authentifizierung eine
E-Mail an den Nutzer, welche einen Link mit einem jwt-token in der URL. Dieser
signierte Token enthält die E-Mail-Adresse. Bei der Antwort des Servers wird das
token aus der URL gelesen und in ein cookie gespeichert.

Beim einem klick auf diese E-Mail kontrolliert der Server, ob die E-Mail-Adresse
bekannt ist. Wenn nicht, dann kommt der Nutzer auf eine Anmelde-Seite, in dem er
seine Nutzerdaten eingeben kann. Die E-Mail-Adresse dieser Seite ist
entsprechend dem Eintrag in dem jwt-token vorausgefüllt und nicht änderbar. Die
Felder auf der Seite sind:

- Name (Pflichtfeld)
- Früherer Name (Optional)
- Möchte Informationen bekommen (Checkbox)
- Werde voraussichtlich teilnehmen (Checkbox)
- Andere dürfen meine Daten sehen (Checkbox)

Wenn die E-Mail-Adresse bereits bekannt ist, dann landet der Nutzer auf einer
Seite, um die Daten (mit ausnahme der E-Mail-Adresse) zu bearbeiten.

Der erste Nutzer der Angelegt wird hat automatisch folgende Flags gesetzt:

- Ist Admin
- Ist überprüft.

Bei allen anderen Nutzern sind diese Daten vorerst False.

Meldet sich ein Admin an, dann wird ihm eine Seite mit allen Nutzern angezeigt.
Andere Nutzer sehen nur die Teilnehmer, die ausgewählt haben, dass sie von
anderen gesehen werden dürfen und überprüft sind. Ein Nutzer, der nicht
überprüft ist, darf keine anderen Daten sehen.

Ein Admin hat das Recht andere Nutzer zu bearbeiten. Auch die E-Mail-Adresse.
Insbesondere kann (nur) ein Admin die Flags "Admin" und "ist überprüft" ändern.


## Technik

Der Server soll mit Go implementiert werden. Als Datenbank wird Sticky
verwendet, eine von mir selbst entwickelte In-Memory-Datenbank, welche Events
speichert.

Die Client-Seite soll durch HTMX mit möglichst wenig Java-Script umgesetzt
werden. Als Template-Engin soll [templ](github.com/a-h/templ) verwendet werden.

Der komplette Server, inklusive des kompletten Client-Codes (html, css etc),
soll in ein Binary kompiliert werden, und zwar mit "CGO_ENABLED=0", so dass es
statisch gelinkt wird.
