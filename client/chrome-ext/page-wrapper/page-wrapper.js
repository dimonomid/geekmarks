'use strict';

(function(exports){

  var uri = new URI(window.location.href);
  var queryParams = uri.search(true);
  var srcDir;
  var htmlPage = undefined;
  var moduleName = undefined;
  switch (queryParams.page) {
    case "get-bookmark":
      srcDir = chrome.extension.getURL("/common/webui/get-bookmark");
      htmlPage = "get-bookmark.html";
      moduleName = "gmGetBookmark";
      break;
    case "edit-bookmark":
      srcDir = chrome.extension.getURL("/common/webui/edit-bookmark");
      htmlPage = "edit-bookmark.html";
      moduleName = "gmEditBookmark";
      break;
    default:
      throw Error("wrong page: " + queryParams.page)
      break;
  }

  var port = chrome.runtime.connect({name: queryParams.port_name});
  var curTab = undefined;

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
              curTab = msg.curTab;
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

    var contentElem = $("#content");

    if (moduleName) {
      contentElem.load(
        srcDir + "/" + htmlPage,
        undefined,
        function() {
          window[moduleName].init(gmClientBridge.create(), contentElem, srcDir);
        }
      );
    } else {
      contentElem.html("wrong page: '" + queryParams.page + "'");
    }
  })

  exports.createGMClient = function() {
    return gmClientBridge.create();
  };

  exports.openPageEditBookmarks = function openPageEditBookmarks(bkmId) {
    port.postMessage({
      type: "cmd", cmd: "openPageEditBookmarks",
      bkmId: bkmId,
      curTab: curTab,
    });
  };

})(typeof exports === 'undefined' ? this['gmPageWrapper']={} : exports);
