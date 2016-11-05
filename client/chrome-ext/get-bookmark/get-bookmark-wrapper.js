//TODO: rename file and module

'use strict';

(function(exports){

  var port = chrome.runtime.connect({name: "queryBkm"});
  var onLoadFunc = undefined;

  //port.postMessage({type: "cmd", cmd: "getCurTab"});

  //TODO: refactor
  var dontNotifyClose = false;

  port.onMessage.addListener(
    function(msg) {
      console.log("got msg:", msg);
      switch (msg.type) {
        case "cmd":
          switch (msg.cmd) {
            case "focus":
              window.focus();
              break;
            case "close":
              window.close();
              dontNotifyClose = true;
              break;
            case "setCurTab":
              console.log("setCurTab:", msg.curTab);
              //alert("hey3: " + JSON.stringify(msg));
              break;
          }
          break;
          //case "response":
          //switch (msg.cmd) {
          //case "getCurTab":
          //alert("hey2: " + JSON.stringify(msg.curTab));
          //break;
          //}
          //break;
      }

    }
  );

  $(window).on("beforeunload", function() { 
    if (!dontNotifyClose) {
      port.postMessage({type: "cmd", cmd: "clearCurTab"});
    }
  })

  $(document).ready(function() {
    $("#content").load(
      chrome.extension.getURL("/common/get-bookmark/get-bookmark.html"),
      undefined,
      function() {
        if (onLoadFunc != undefined) {
          onLoadFunc();
        }
      }
    );
  })

  exports.createGMClient = function() {
    return gmClientBridge.create();
  };

  exports.onLoad = function(f) {
    onLoadFunc = f;
  };

})(typeof exports === 'undefined' ? this['getBookmarkWrapper']={} : exports);
