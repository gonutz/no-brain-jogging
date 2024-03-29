No-Brain Jogging
================

![Screenshot](https://raw.githubusercontent.com/gonutz/no-brain-jogging/master/screenshots/screenshot.png)

This is a port of [my entry](https://ldjam.com/events/ludum-dare/41/no-brain-jogging) for the [Ludum Dare 41](https://ldjam.com/events/ludum-dare/41) game jam to [the pixel game engine](https://github.com/faiface/pixel). The theme was: "Combine two Incompatible Genres".

My two genres are:

- 2D side-scrolling zombie shooter
- educational math game / brain jogging

In this game you solve math calculations to shoot your rifle and kill some zombies. Kill as many as you can before they eat your brains.

The [original version](https://github.com/gonutz/ld41) was written using my [prototype](https://github.com/gonutz/prototype) library. This is a rewrite using [faiface](https://github.com/faiface)'s [pixel game library](https://github.com/faiface/pixel) and [beep audio library](https://github.com/faiface/beep). I wanted to try out these libraries because I intend to use them for the next Ludum Dare game jam.

Build Instructions
==================

This game uses [pixel](https://github.com/faiface/pixel) and [beep](https://github.com/faiface/beep) so please see their instructions for prerequisites to build these depedencies. Once you have set them up, you can simply do:

    go get github.com/gonutz/no-brain-jogging

Navigate to the `no-brain-jogging` directory and run:

    go build

which will create the executable.
