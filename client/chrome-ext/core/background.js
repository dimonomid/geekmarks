
function concatenateInjections(id, ar, scrpt){
  if( typeof scrpt !== 'undefined' ) ar = ar.concat([scrpt]);

  var i = ar.length;
  var idx = 0 ;

  (function (){
    var that = arguments.callee;
    idx++;
    if(idx <= i){
      var f = ar[idx-1];
      var func = chrome.tabs.executeScript;
      if (f.slice(-4) == ".css") {
        func = chrome.tabs.insertCSS;
      }
      func(id, { file: f }, function(){ that(idx);} );
    }
  })();
}

//chrome.commands.onCommand.addListener(function(command) {
  //console.log('onCommand event received for message: ', command);

  //concatenateInjections(null, [
    //"vendor/jquery-3.1.1.min.js",
    //"vendor/jquery-ui/jquery-ui.min.js",
    //"vendor/jquery-ui/jquery-ui.min.css",
    //"vendor/jquery-ui/jquery-ui.structure.min.css",
    //"vendor/jquery-ui/jquery-ui.theme.min.css",
    //"js/dialog.js",
  //]);
//});

var clientInst = gmClient.create("localhost:4000", "alice", "alice");

var queryBkmContext = {
  tab: undefined,
  port: undefined,
}

chrome.commands.onCommand.addListener(function(command) {
  //window.open("html/get-bookmark-wrapper.html", "extension_popup", "width=300,height=400,status=no,scrollbars=yes,resizable=no");
  var curTab;

  console.log("got command:", command);
  chrome.tabs.query({active: true, currentWindow: true}, function(arrayOfTabs) {
    switch (command) {
      case "query-bookmark":
        {
          // TODO: better check of whether it's the window of this extension
          if (arrayOfTabs[0].url.slice(6) === "chrome") {
            console.log("url chrome: ignoring")
            return;
          }

          // since only one tab should be active and in the current window at once
          // the return variable should only have one entry
          curTab = arrayOfTabs[0];
          //alert(JSON.stringify(curTab));

          queryBkmContext.tab = curTab;

          if (queryBkmContext.port === undefined) {
            chrome.windows.create({
              url: "/page-wrapper/page-wrapper.html?page=get-bookmark",
              //left: 100,
              //top: 100,
              width: 700,
              height: 400,
              type: "popup",
            });
          } else {
            console.log("refocusing");
            queryBkmContext.port.postMessage(
              {type: "cmd", cmd: "setCurTab", curTab: queryBkmContext.tab}
            );

            queryBkmContext.port.postMessage({type: "cmd", cmd: "focus"});
          }
        }
        break;
    }

  });

});

//var ports = {};

chrome.runtime.onConnect.addListener(
  function(port) {
    console.log("connected, port name:", port.name);

    switch (port.name) {
      case "queryBkm":
        queryBkmContext.port = port;
        queryBkmContext.port.postMessage(
          {type: "cmd", cmd: "setCurTab", curTab: queryBkmContext.tab}
        );

        port.onMessage.addListener(
          function(msg) {
            console.log("got msg:", msg);
            switch (msg.type) {
              case "cmd":
                switch (msg.cmd) {
                  //case "getCurTab":
                    //console.log("sending curTab")
                    //port.postMessage({type: "response", cmd: msg.cmd, curTab: curTab});
                    //break;
                  case "clearCurTab":
                    queryBkmContext.tab = undefined;
                    queryBkmContext.port = undefined;
                    break;
                }
                break;
            }
          }
        );
        break;

      case "gmclient-bridge":
        port.onMessage.addListener(
          function(msg) {
            console.log("got msg:", msg);
            switch (msg.type) {
              case "cmd":
                switch (msg.cmd) {
                case "sendViaGMClient":
                  var func = clientInst[msg.funcName];
                  msg.args.push(function(resp) {
                    port.postMessage(
                      {type: "cmd", cmd: "gmClientResp", resp: resp, id: msg.id}
                    );
                  })
                  func.apply(undefined, msg.args);
                  break;
                }
                break;
            }
          }
        );
        break;
      default:
        alert("unknown port name: " + port.name);
        break;
    }

    //if (port.name in ports) {
      //ports[port.name].port.postMessage({type: "cmd", cmd: "close"});
      //delete ports[port.name];
    //}

    //ports[port.name] = {
      //port: port,
    //};

    //port.onMessage.addListener(
      //function(msg) {
        //console.log("got msg:", msg);
        //switch (msg.type) {
          //case "cmd":
            //switch (msg.cmd) {
              //case "getCurTab":
                //console.log("sending curTab")
                //port.postMessage({type: "response", cmd: msg.cmd, curTab: curTab});
                //break;
              //case "clearCurTab":
                //curTab = undefined;
                //break;
            //}
            //break;
        //}
      //}
    //);
  }
);

//chrome.runtime.onMessage.addListener(
  //function(request, sender, sendResponse) {
    //console.log(
      //sender.tab ?
                //"from a content script:" + sender.tab.url :
                //"from the extension"
    //);
    //switch (request.cmd) {
      //case "getCurTab":
        //sendResponse({curTab: curTab});
        //break;
      //case "clearCurTab":
        //curTab = undefined;
        //break;
    //}
  //}
//);
