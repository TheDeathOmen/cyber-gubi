# cyber-gubi - Guaranteed unconditional basic income digital currency

## Notes

+ Add GDPR compliance notice for collection of biometrics data - not needed since we don't store it anymore, it's in the localStorage of the user
+ Check for PAD models
+ Check all Uint8Array values are correct with web crypto API

## Features

+ Not a crypto currency - not tradeable on exchanges, not based on blockchain
+ Pseudonymous - your only identificator is your peer ID which is mapped to your IP address
+ No data collection - no registration, no biometrics, no personal data
+ Natural supply cap - the amount of tokens is limited to the amount of living people receiving them
+ Natural burn rate - when a beneficiary dies the tokens belonging to that person are out of circulation
+ No transfer - tokens can only be used for purchases
+ No inheritance - tokens can not be inherited or moved between people
+ No exchange - tokens are not listed on exchanges and can not be traded
+ Inflation auto-corrector - real-time analysis of price fluctuations and adjustment of income

## Road to adoption

+ Pricing starting point maps cyber-gubi 1 to 1 with your local currency - this is purely for reference since it can not be exchanged
+ When we have enough influence and purchasing power expressed in number of people on basic income and amount of tokens received we can ask retailers to accept it
+ Once a big enough commercial network wants to use it due to the amount of potential customers having it there is a push towards states to accept it as a form of currency to pay taxes in
+ A free-market economy sets in and starts regulating internal pricing independently of other currencies
+ The value of cyber-gubi is defined by each retailer setting a price on own products
+ Wages in cyber-gubis are adjusted based on arbitrage between product prices and profit
+ All-in-one depots emerge where you pay a monthly fee and use and return instead of own

## FAQ

+ Why not list it on exchanges and pay taxes in the accepted local currency?
    + It will immediately be speculated with and inflated/deflated.
+ How to prevent multiple wallets via different devices?
    + Try https://github.com/go-webauthn/webauthn
+ How to adjust supply/burn rate based on increase of prices?
    + Keep track of all price changes for all goods on a monthly basis and re-index the basic income amount according to inflation
+ Why is there no mobile version?
    + App stores are centralized and can take down the app anytime. You can use a mini laptop with Linux instead.

## Credits

+ Modified [Wallet Layout](https://codepen.io/surendharnagarajan/pen/eoKOLL)

Copyright (c) 2025 by Surendhar Nagarajan (https://codepen.io/surendharnagarajan/pen/eoKOLL)

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.