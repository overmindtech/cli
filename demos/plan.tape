Output demos/plan.gif
# Output demos/plan.mp4

Set Margin 20
Set MarginFill "#7a70eb" # use Dark.BgMain
Set BorderRadius 10

Hide
Type "cd tmp"
Enter
Type "export PATH=$PWD:$PATH"
Enter
Type "clear"
Enter
Show

Type@10ms "overmind terraform plan"
Enter
Sleep 2
Down
Sleep 500ms
Down
Sleep 1
Enter
Type "sso-dogfood"
Sleep 1
Enter
Sleep 20

# Ctrl+c
# Sleep 10
