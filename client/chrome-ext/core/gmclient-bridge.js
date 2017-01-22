'use strict';

(function(exports){

  function _createGMClientLoggedIn(){

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
                    console.log('got response from the bridge:', msg)
                    pr.cb.apply(undefined, msg.respArgs);
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
        type: "cmd", cmd: "sendViaGMClient", funcName: funcName,
        args: args,
      };

      pendingRequests[sID] = {
        cb: cb,
      };
      port.postMessage(msg)
    }

    function createWrapperFunc(name) {
      return function() {
        var args = [];
        var cb = undefined;

        for (var i = 0; i < arguments.length - 1 /* cb */; i++) {
          args.push(arguments[i]);
        }
        if (arguments.length > 0) {
          cb = arguments[arguments.length - 1];
        }

        send(name, args, cb);
      }
    }

    return {
      onConnected: createWrapperFunc("onConnected"),
      getTagsTree: createWrapperFunc("getTagsTree"),
      getTagsByPattern: createWrapperFunc("getTagsByPattern"),
      getTag: createWrapperFunc("getTag"),
      addTag: createWrapperFunc("addTag"),
      updateTag: createWrapperFunc("updateTag"),
      deleteTag: createWrapperFunc("deleteTag"),
      getTaggedBookmarks: createWrapperFunc("getTaggedBookmarks"),
      getBookmarksByURL: createWrapperFunc("getBookmarksByURL"),
      getBookmarkByID: createWrapperFunc("getBookmarkByID"),
      addBookmark: createWrapperFunc("addBookmark"),
      updateBookmark: createWrapperFunc("updateBookmark"),

      //getTagsByPattern: function getTagsByPattern(pattern, allowNew, cb) {
      //send("getTagsByPattern", [pattern, allowNew], cb)
      //},
      //getTaggedBookmarks: function getTaggedBookmarks(tagIDs, cb) {
      //send("getTaggedBookmarks", [tagIDs], cb)
      //},
    };
  }

  exports.createGMClientLoggedIn = function() {
    return new Promise(function(resolve, reject) {
      gmClientFactory.getLocalData().then(function(data) {
        if (data !== undefined && "token" in data) {
          resolve(_createGMClientLoggedIn());
        } else {
          resolve(null);
        }
      })
    });
  };

})(typeof exports === 'undefined' ? this['gmClientBridge']={} : exports);
