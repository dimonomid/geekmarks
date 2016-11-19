'use strict';

(function(exports){

  exports.create = function(server, user, password){

    var msgID = 0;
    var pendingRequests = {};
    var isConnected = false;
    var onConnectedCB = undefined;
    var ws = new WebSocket(
      "ws://" + user + ":" + password + "@" + server + "/api/my/wsconnect"
    );

    ws.onopen = function() {
      console.log("Connection opened");
      isConnected = true;
      if (onConnectedCB) {
        onConnectedCB();
      }
    };
    ws.onclose = function() {
      console.log("Connection closed");
      isConnected = false;
      // TODO: reconnect
    };

    ws.onmessage = function(evt) {
      console.log("ws event:", evt);
      var msg = JSON.parse(evt.data);
      var sID = String(msg.id);
      if (sID in pendingRequests) {
        var pr = pendingRequests[sID];
        if (typeof(pr) === 'object') {
          pr.cb(msg.status, msg.body);
        }
        delete pendingRequests[sID];
      }
    };

    function send(msg, cb) {
      msgID++;
      var sID = String(msgID)
      if (sID in pendingRequests) {
        throw Error("should never happen");
      }

      msg.id = msgID;

      pendingRequests[sID] = {
        cb: cb,
      };
      ws.send(JSON.stringify(msg));
    }

    function onConnected(invokeIfAlreadyConnected, cb) {
      onConnectedCB = cb;
      if (onConnectedCB && isConnected && invokeIfAlreadyConnected) {
        onConnectedCB();
      }
    }

    function getTagsByPattern(pattern, allowNew, cb) {
      console.log("getTagsByPattern is called, pattern:", pattern);
      var values = {
        shape: "flat",
        pattern: pattern,
      };
      if (allowNew) {
        values.allow_new = "1";
      }
      send({
        path: "/tags",
        method: "GET",
        values: values,
      }, cb);
    }

    function getTaggedBookmarks(tagIDs, cb) {
      console.log("getTaggedBookmarks is called, tagIDs:", tagIDs);
      send({
        path: "/bookmarks",
        method: "GET",
        values: {
          tag_id: tagIDs,
        }
      }, cb);
    }

    function getBookmarkByID(bookmarkID, cb) {
      console.log("getBookmarkByID is called, bookmarkID:", bookmarkID);
      send({
        path: "/bookmarks/" + bookmarkID,
        method: "GET"
      }, cb);
    }

    function saveBookmark(bookmarkID, bkmData, cb) {
      console.log("saveBookmark is called, bookmarkID:", bookmarkID, ", bkmData:", bkmData);
      send({
        path: "/bookmarks/" + bookmarkID,
        body: {
          url: bkmData.url,
          title: bkmData.title,
          comment: bkmData.comment,
          tagIDs: bkmData.tagIDs,
          //TODO: newTagNames
        },
        method: "PUT"
      }, cb);
    }

    return {
      onConnected: onConnected,
      getTagsByPattern: getTagsByPattern,
      getTaggedBookmarks: getTaggedBookmarks,
      getBookmarkByID: getBookmarkByID,
      saveBookmark: saveBookmark,
    };

  };

})(typeof exports === 'undefined' ? this['gmClient']={} : exports);
