dgVoice
====
[![GoDoc](https://godoc.org/github.com/bwmarrin/dgvoice?status.svg)](https://godoc.org/github.com/bwmarrin/dgvoice) [![Go report](http://goreportcard.com/badge/bwmarrin/dgvoice)](http://goreportcard.com/report/bwmarrin/dgvoice) [![Build Status](https://travis-ci.org/bwmarrin/dgvoice.svg?branch=master)](https://travis-ci.org/bwmarrin/dgvoice) [![Discord Gophers](https://img.shields.io/badge/Discord%20Gophers-%23info-blue.svg)](https://discord.gg/0f1SbxBZjYq9jLBk)

dgVoice is a [Go](https://golang.org/) package that provides an example of 
adding opus audio and play file support for [DiscordGo](https://github.com/bwmarrin/discordgo).

* You must use the current develop branch of Discordgo
* You must have ffmpeg in your path and Opus libs already installed.

This code should be considered just a proof of concept, or an example, of 
accomplishing this task and not a defacto standard. 

Please send feedback on any performance improvements that can be made for 
sound quality, stability, or efficiency.


**For help with this package or general Go discussion, please join the [Discord 
Gophers](https://discord.gg/0f1SbxBZjYq9jLBk) chat server.**

## Getting Started

### Installing

This assumes you already have a working Go environment, if not please see
[this page](https://golang.org/doc/install) first.

```sh
go get github.com/bwmarrin/dgvoice
```

# Usage Example
See example folder
