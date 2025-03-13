Output e2e/plan.mp4

Set Width 1920
Set Height 1080

Set FontSize 12
Set CursorBlink false

Hide
Type "export PATH=$PWD:$PATH"
Enter
Type "clear"
Enter
Show

Type@1ms "overmind terraform plan -- -out tfplan"
Enter
Sleep 2
Enter
Sleep 80
Screenshot e2e/plan.png
