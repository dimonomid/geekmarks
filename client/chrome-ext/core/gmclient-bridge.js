'use strict';

(function(exports){

  exports.create = function(){

    var msgID = 0;
    var pendingRequests = {};
    var port = chrome.runtime.connect({name: "gmclient-bridge"});

    port.onMessage.addListener(
      function(msg) {
        switch (msg.type) {
          case "cmd":
            switch (msg.cmd) {
              case "gmClientResp":
                var sID = String(msg.id);
                if (sID in pendingRequests) {
                  var pr = pendingRequests[sID];
                  if (typeof(pr) === 'object') {
                    //console.log('hey:', msg)
                    pr.cb(msg.resp);
                  }
                  delete pendingRequests[sID];
                }
                break;
            }
            break;
        }
      }
    )

    function send(funcName, args, cb) {
      msgID++;
      var sID = String(msgID)
      if (sID in pendingRequests) {
        throw Error("should never happen");
      }

      var msg = {
        id: msgID,
        type: "cmd", cmd: "sendViaGMClient", funcName: "getTagsByPattern",
        args: args,
      };

      pendingRequests[sID] = {
        cb: cb,
      };
      port.postMessage(msg)
    }

    return {
      getTagsByPattern: function getTagsByPattern(pattern, cb) {
        send("getTagsByPattern", [pattern], cb)
      },
    };

  };

})(typeof exports === 'undefined' ? this['gmClientBridge']={} : exports);
