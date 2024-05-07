# backgommon
Backgommon is a backtesting and simulation framework for trading strategies, written in pure go. It aims to be fast, flexible and easy to use.

#### NOTE: code in this repo is being transferred from a private repo (which has my own strategy, some secret keys, and PII data) I will be cleaning the code and removing the PII data and secret keys. Please be patient while I transfer the code, make it public and write some documentation.

## Why a(nother) backtesting framework?
I created this project primarily to learn go and backtest and implement some trading strategies. When I searched for backtesting frameworks, across all languages, most of them (all?) were about testing strategies on a single ticker/scrip/asset. What I wanted was a framework that could backtest strategy on portfolio of assets and buy/sell/manage a portfolio. And thus backgommon was born.

> Note: number of assets in portfolio can as well be 1, so you can use it for single asset backtesting, not that you have to stick to multi-asset portfolio only.

Initially it was made as a closed source application and was highly coupled with my own strategies, the framework itself was simple and fast but it was not very flexible. I had plans to make it open source since the beginning but couldn't since it was highly coupled with my strategies. Now I have decided to "kind of" write it from scratch making it open source, clean, flexible, and easy to use.

## WIP
As I mentioned this project is a work in progress, I will be adding more features and improving the codebase over time. If you would like to contribute, please feel free to open a pull request. Also as one of the aims of this project, for me, was to learn golang, please feel free to critique the codebase and suggest improvements!

## What backgommon is not?
Backgommon is not a high frequency trading framework, it is not designed for HFT. You would probably want to write your HFT strategies in C++ or Rust or C++ or Zig or C++ or Java maybe?

Having said that, I have come to realize that go is pretty fast and maybe with some optimizations and good system design, backgommon can be used for HFT as well. And having said that, I **don't** have plans to add HFT support in the foreseeable future.

## Not just backtesting...
... but also live trading. I have plans to add live trading support as well. I have already written a web server for live strategy and portfolio monitoring, which I will be cleaning and adding to this repo.

Live trading system aims to be a simple plug-n-play solution, where you can plug in your input feed (from your broker or any other source) via multiple interfaces and let backgommon take care of the rest. Generated signals can then be fed to your broker/alert systems again via multiple interfaces.

## Features
- TODO
- yeah i need to classify what would count as a feature and list it down here
- so thats a TODO

## TODO
- Adding more technical indicator
- Monte Carlo simulation
- Graph plotting
- Web server for live strategy and portfolio monitoring (copy pasting and cleaning from a crappy private repo)
- MOREEE Documentation
