'use strict';

(function(exports){

  exports.getOptions = function(cb) {
    chrome.storage.sync.get({
      server: "geekmarks.dmitryfrank.com",
      serverSSL: true,
    }, cb);
  }

  exports.setOptions = function(opts, cb) {
    chrome.storage.sync.set(opts, function() {
      cb();
      if ("server" in opts || "serverSSL" in opts) {
        // we need to reconnect
        // TODO: reconnect more gently
        chrome.runtime.reload();
      }
    });
  }

})(typeof exports === 'undefined' ? this['gmOptions']={} : exports);
