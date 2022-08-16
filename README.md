# waybar-eyes

This is a waybar applet to help you keeping your eyes.

This will try to detect your face and indicate you in waybar if you are stuck to your screen for too much time.
This add a new eye in the bar every 15 minutes if a face is detected.
This eyes number in the bar will decrease if you take a pause.

## Requirements

- golang >= 1.18
- opencv4
- qt5-base

## Build

```shell
make
```

## Waybar config

```json
...
"modules-right": [
  "custom/eyes",
],
...
"custom/eyes": {
    "exec": "cat ~/.cache/waybar-eyes.json",
    "interval": 5,
    "return-type": "json",
    "on-click": "pkill -f -SIGUSR1 waybar-eyes",
},
...
```

## Resources

You can test differents detection models from here

- https://github.com/opencv/opencv/tree/master/data/haarcascades
- https://github.com/kipr/opencv/tree/master/data/haarcascades
