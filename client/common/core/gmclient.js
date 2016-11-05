'use strict';

(function(exports){

  exports.create = function(server, user, password){

    var msgID = 0;
    var pendingRequests = {};
    var ws = new WebSocket(
      "ws://" + user + ":" + password + "@" + server + "/api/my/wsconnect"
    );

    ws.onopen = function() { console.log("Connection opened"); };
    ws.onclose = function() { console.log("Connection closed"); };
    ws.onmessage = function(evt) {
      console.log("ws event:", evt);
      var msg = JSON.parse(evt.data);
      var sID = String(msg.id);
      if (sID in pendingRequests) {
        var pr = pendingRequests[sID];
        if (typeof(pr) === 'object') {
          pr.cb(msg.body);
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

    function getTagsByPattern(pattern, cb) {
      console.log("getTagsByPattern is called, pattern:", pattern);
      send({
        path: "/tags",
        method: "GET",
        values: {
          shape: "flat",
          pattern: pattern,
        }
      }, cb);
    }

    return {
      getTagsByPattern: getTagsByPattern,
    };

  };

})(typeof exports === 'undefined' ? this['gmClient']={} : exports);
