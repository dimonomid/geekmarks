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
