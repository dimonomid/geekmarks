'use strict';

(function(exports){

  exports.create = function(opts){

    opts = $.extend({}, {
      server: "",

      // All these callbacks receive an instance of gmClient as `this`.
      setLocalData: function(data){},
      getLocalData: function(){},
      launchWebAuthFlow: function(url){},
    }, opts);

    console.log('opts', opts)

    function handleResp(status, resp, resolve, reject, cb) {
      if (status === 200) {
        resolve(resp);
      } else {
        reject(resp);
      }

      if (cb) {
        cb(status, resp);
      }
    }

    function getOAuthClientID(provider, cb) {
      return new Promise(function(resolve, reject) {
        var xhr = new XMLHttpRequest();
        xhr.onload = function() {
          var resp = JSON.parse(xhr.responseText);
          handleResp(xhr.status, resp, resolve, reject, cb);
        }
        xhr.onerror = function(e) {
          reject({status: e.target.status});
        }
        xhr.open(
          "GET",
          "http://" + opts.server + "/api/auth/" + provider + "/client_id",
          true
        );
        xhr.send(null);
      });
    }

    function authenticate(provider, redirectURI, code, cb) {
      return new Promise(function(resolve, reject) {
        var xhr = new XMLHttpRequest();
        xhr.onload = function() {
          var resp = JSON.parse(xhr.responseText);
          handleResp(xhr.status, resp, resolve, reject, cb);
        }
        xhr.onerror = function(e) {
          reject({status: e.target.status});
        }
        var url = URI("http://" + opts.server + "/api/auth/" + provider + "/authenticate")
          .addSearch("code", code)
          .addSearch("redirect_uri", redirectURI)
          .toString();
        xhr.open(
          "POST",
          url,
          true
        );
        xhr.send(null);
      });
    }

    function onAuthenticated(resp) {
      return opts.setLocalData.call(this, resp).then(function() {
        return createGMClientLoggedIn();
      });
    }

    function createGMClientLoggedIn() {
      return new Promise(function(resolve, reject) {
        opts.getLocalData.call(this).then(function(data) {
          if (data !== undefined && "token" in data) {
            resolve(_createGMClientLoggedIn(data.token));
          } else {
            resolve(null);
          }
        })
      });
    }

    function login(provider) {
      return opts.launchWebAuthFlow.call(this, provider);
    }

    function logout() {
      return opts.setLocalData.call(this, {});
    }

    function _createGMClientLoggedIn(token) {
      var msgID = 0;
      var pendingRequests = {};
      var isConnected = false;
      var onConnectedCB = undefined;
      var artificialDelay = 0;
      var ws = new WebSocket(
        "ws://" + opts.server + "/api/my/wsconnect?token=" + encodeURIComponent(token)
      );

      if (opts.server.substring(0, 9) === "localhost") {
        artificialDelay = 150;
      }

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

        var f = function() {
          ws.send(JSON.stringify(msg));
        };

        if (artificialDelay == 0) {
          f();
        } else {
          setTimeout(f, artificialDelay);
        }
      }

      function onConnected(invokeIfAlreadyConnected, cb) {
        onConnectedCB = cb;
        if (onConnectedCB && isConnected && invokeIfAlreadyConnected) {
          onConnectedCB();
        }
      }

      function getTagsTree(cb) {
        // TODO: support subpath, support no-subtags
        console.log("getTagsTree is called");
        send({
          path: "/tags",
          method: "GET",
        }, cb);
      }

      function getTag(tagPathOrID, cb) {
        // TODO: support subpath, support no-subtags
        console.log("getTag is called");
        send({
          path: ["/tags", tagPathOrID].join("/"),
          method: "GET",
          shape: "single",
        }, cb);
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

      function addTag(parentTagPath, data, cb) {
        console.log("addTag is called:", parentTagPath, data);
        send({
          path: ["/tags", parentTagPath].join("/"),
          method: "POST",
          body: {
            names: data.names,
            description: data.description,
            createIntermediary: data.createIntermediary,
          },
        }, cb);
      }

      function updateTag(tagPathOrID, tagData, cb) {
        console.log("updateTag is called, tagPathOrID:", tagPathOrID, ", tagData:", tagData);
        send({
          path: ["/tags", tagPathOrID].join("/"),
          body: {
            names: tagData.names,
            description: tagData.description,
            parentTagID: Number(tagData.parentTagID),
          },
          method: "PUT"
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

      function getBookmarksByURL(url, cb) {
        console.log("getBookmarksByURL is called, url:", url);
        send({
          path: "/bookmarks",
          method: "GET",
          values: {
            url: [url],
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

      function addBookmark(bkmData, cb) {
        console.log("addBookmark is called:", ", bkmData:", bkmData);
        send({
          path: "/bookmarks",
          body: {
            url: bkmData.url,
            title: bkmData.title,
            comment: bkmData.comment,
            tagIDs: bkmData.tagIDs,
          },
          method: "POST"
        }, cb);
      }

      function updateBookmark(bookmarkID, bkmData, cb) {
        console.log("updateBookmark is called, bookmarkID:", bookmarkID, ", bkmData:", bkmData);
        send({
          path: "/bookmarks/" + bookmarkID,
          body: {
            url: bkmData.url,
            title: bkmData.title,
            comment: bkmData.comment,
            tagIDs: bkmData.tagIDs,
          },
          method: "PUT"
        }, cb);
      }

      return {
        onConnected: onConnected,
        getTagsTree, getTagsTree,
        getTagsByPattern: getTagsByPattern,
        getTag: getTag,
        addTag: addTag,
        updateTag, updateTag,
        getTaggedBookmarks: getTaggedBookmarks,
        getBookmarksByURL: getBookmarksByURL,
        getBookmarkByID: getBookmarkByID,
        addBookmark: addBookmark,
        updateBookmark: updateBookmark,
      };
    }

    return {
      createGMClientLoggedIn: createGMClientLoggedIn,
      login: login,
      logout: logout,
      getOAuthClientID: getOAuthClientID,
      authenticate: authenticate,
      onAuthenticated: onAuthenticated,
    };

  };

})(typeof exports === 'undefined' ? this['gmClient']={} : exports);
