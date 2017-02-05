var clientInst = undefined;
var clientLoggedInInst = undefined;
var clientLoggedInInstPromise = undefined;

gmClientFactory.create(false /* without bridge */).then(function(inst) {
  clientInst = inst;

  // Initialize clientLoggedInInstPromise and, when it is resolved,
  // clientLoggedInInst
  getClientLoggedInInst();

  pagesCtx = {};

  chrome.commands.onCommand.addListener(function(command) {
    //window.open("html/find-bookmark-wrapper.html", "extension_popup", "width=300,height=400,status=no,scrollbars=yes,resizable=no");
    var curTab;

    console.log("got command:", command);
    chrome.tabs.query({active: true, currentWindow: true}, function(arrayOfTabs) {

      // TODO: better check of whether it's the window of this extension
      if (arrayOfTabs[0].url.slice(6) === "chrome") {
        console.log("url chrome: ignoring")
        return;
      }

      switch (command) {
        case "query-bookmark":
          {
            // since only one tab should be active and in the current window at once
            // the return variable should only have one entry
            curTab = arrayOfTabs[0];

            openOrRefocusPageWrapper("findBookmark", "page=find-bookmark", curTab);
          }
          break;
        case "add-bookmark":
          {
            // since only one tab should be active and in the current window at once
            // the return variable should only have one entry
            curTab = arrayOfTabs[0];

            openPageAddBookmark(curTab);
          }
          break;
        case "tags-tree":
          {
            openOrRefocusPageWrapper("tagsTree", "page=tags-tree", curTab);
          }
          break;
        case "login-logout":
          {
            openOrRefocusPageWrapper("loginLogout", "page=login-logout", curTab);
          }
          break;
      }

    });

  });

  chrome.runtime.onConnect.addListener(
    function(port) {
      console.log("connected, port name:", port.name);

      switch (port.name) {
        case "gmclient-bridge":
          port.onMessage.addListener(
            function(msg) {
              console.log("got msg:", msg);
              switch (msg.type) {
                case "cmd":
                  switch (msg.cmd) {
                    case "sendViaGMClient":
                      getClientLoggedInInst().then(function(loggedInInst) {
                        if (loggedInInst) {
                          var func = loggedInInst[msg.funcName];
                          msg.args.push(function(/*arbitrary args*/) {
                            /*
                          * We have to copy things from `arguments` to a real
                          * array, in order for the apply() in the gmclient-bridge
                          * to work correctly
                          */
                            var args = [];
                            for (var i = 0; i < arguments.length; i++) {
                              args.push(arguments[i]);
                            }
                            console.log('sending resp back:', args)
                            port.postMessage(
                              {type: "cmd", cmd: "gmClientResp", respArgs: args, id: msg.id}
                            );
                          })
                          func.apply(loggedInInst, msg.args);
                        } else {
                          /*
                        * Asked to call some gmClientLoggedIn function while
                        * we're not logged in: actually it should not happen.
                        */
                          console.log("failed to send via gmclient: not logged in");
                        }
                      })
                      break;
                  }
                  break;
              }
            }
          );
          break;

        default:
          if (pagesCtx[port.name].port !== undefined) {
            throw Error("port for " + port.name + " already exists when new port connection is created");
          }

          pagesCtx[port.name].port = port;
          setCurTab(port.name);

          port.onMessage.addListener(
            function(msg) {
              console.log("got msg in port", port, ":", msg);
              switch (msg.type) {
                case "cmd":
                  switch (msg.cmd) {
                    case "clearCurTab":
                      delete pagesCtx[port.name]
                      break;

                    case "openPageGetBookmark":
                      openPageGetBookmark(msg.curTab);
                      break;

                    case "openPageTagsTree":
                      openPageTagsTree(msg.curTab);
                      break;

                    case "openPageEditBookmarks":
                      openPageEditBookmarks(msg.bkmId, msg.curTab);
                      break;

                    case "openPageAddBookmark":
                      openPageAddBookmark(msg.curTab);
                      break;

                    case "openPageEditTag":
                      openPageEditTag(msg.tagId, msg.curTab);
                      break;

                    case "openPageLogin":
                      openPageLogin(msg.backFunc, msg.backArgs, msg.curTab);
                      break;
                  }
                  break;
              }
            }
          );

          break;
      }

    }
  );

});

// Returns a promise which gets resolved to a gmClientLoggedIn instance or, if
// the user is not logged in, to null
function getClientLoggedInInst() {
  return new Promise(function(resolve, reject) {
    if (clientLoggedInInst) {
      // Already logged in: resolve immediately
      resolve(clientLoggedInInst);
    } else {
      // Not logged in. First of all, let's see if we're already waiting for
      // the instance, and if not, then start waiting
      if (clientLoggedInInstPromise === undefined) {
        clientLoggedInInstPromise = clientInst.createGMClientLoggedIn();
      }

      // Now, clientLoggedInInstPromise is a valid promise. When it gets
      // resolved, remember the clientLoggedInInst instance, and clear the
      // promise.
      clientLoggedInInstPromise.then(function(v) {
        clientLoggedInInst = v;
        clientLoggedInInstPromise = undefined;
      });

      // Resolve to a promise
      resolve(clientLoggedInInstPromise);
    }
  })
}

function openPageWrapper(queryString) {
  chrome.windows.create({
    url: "/page-wrapper/page-wrapper.html?" + queryString,
    width: 700,
    height: 400,
    type: "popup",
  });
}

function openOrRefocusPageWrapper(portName, queryString, curTab) {
  if (!(portName in pagesCtx)) {
    console.log("opening", portName, curTab);
    pagesCtx[portName] = {
      port: undefined,
      tab: curTab,
    };
    openPageWrapper(queryString + '&port_name=' + portName);
  } else {
    console.log("refocusing", portName, curTab);
    pagesCtx[portName].tab = curTab;
    setCurTab(portName);
    pagesCtx[portName].port.postMessage({type: "cmd", cmd: "focus"});
  }
}

function setCurTab(portName) {
  pagesCtx[portName].port.postMessage(
    {type: "cmd", cmd: "setCurTab", curTab: pagesCtx[portName].tab}
  );
}

function openPageGetBookmark(curTab) {
  openOrRefocusPageWrapper("findBookmark", "page=find-bookmark", curTab);
}

function openPageTagsTree(curTab) {
  openOrRefocusPageWrapper("tagsTree", "page=tags-tree", curTab);
}

function openPageEditBookmarks(bkmId, curTab) {
  openOrRefocusPageWrapper(
    "editBookmark-" + bkmId, "page=edit-bookmark&bkm_id=" + bkmId, curTab
  );
}

function openPageAddBookmark(curTab) {
  openOrRefocusPageWrapper(
    "addBookmark", "page=edit-bookmark&bkm_id=0", curTab
  );
}

function openPageEditTag(tagId, curTab) {
  openOrRefocusPageWrapper(
    "editTag-" + tagId, "page=edit-tag&tag_id=" + tagId, curTab
  );
}

function openPageLogin(backFunc, backArgs, curTab) {
  openOrRefocusPageWrapper(
    "loginLogout",
    "page=login-logout&backFunc=" + encodeURIComponent(backFunc) +
    "&backArgs=" + encodeURIComponent(JSON.stringify(backArgs))
    ,
  curTab
  );
}
