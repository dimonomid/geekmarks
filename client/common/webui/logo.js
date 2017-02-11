// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.
//
// NOTE: unlike the backend code, which is thoroughly tested and is very
// stable, this (client) code is only a proof of concept so far. It's just
// barely good enough.

'use strict';

(function(exports){

  function getLogoData() {
    var d = "";

    d += "                     ██                              ██           \n";
    d += "                     ██                              ██           \n";
    d += "▄█▀▀██▀▄█▀▀█▄ ▄█▀▀█▄ ██ ▄▀▀ ██▄▀█▄▄▀█▄ ▄█▀▀█▄  ██▄▀▀ ██ ▄▀▀ ▄█▀▀█▄\n";
    d += "██  ██ ██▄▄██ ██▄▄██ ███▄   ██  ██  ██  ▄▄▄██  ██    ███▄   ▀█▄▄▄ \n";
    d += "██  ██ ██  ▄▄ ██  ▄▄ ██ ▀█▄ ██  ██  ██ ██  ██  ██    ██ ▀█▄ ▄▄  ██\n";
    d += "▄▀▀▀▀   ▀▀▀▀   ▀▀▀▀  ▀▀   ▀ ▀▀  ▀▀  ▀▀  ▀▀▀▀▀▀ ▀▀    ▀▀   ▀  ▀▀▀▀ \n";
    d += "▄▀▀▀▀▄ ____________________________________________B_E_T_A________\n";
    d += "▀▄▄▄▄▀                                                            \n";

    return d;
  }

  function getLogoDataHtml() {
    return getLogoData().replace(/ /g, '&nbsp;').replace(/\n/g, '<br/>');
  }

  exports.getLogoData = getLogoData;
  exports.getLogoDataHtml = getLogoDataHtml;

})(typeof exports === 'undefined' ? this['gmLogo']={} : exports);
