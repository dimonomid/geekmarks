'use strict';

(function(exports){

  var contentElem = undefined;
  var moveDialog = undefined;
  var delDialog = undefined;
  var gmClientLoggedIn = undefined;

  var rootTagKey = undefined;
  var keyToTag = {};

  function init(_gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;

    // Setup tag move dialog {{{
    moveDialog = contentElem.find('#move_dialog')
    moveDialog.dialog({
      dialogClass: "no-close",
      buttons: [
        {
          id: "move-button-ok",
          text: "Move!",
          click: function() {
            var self = this;

            var node = $(this).data("node");
            var data = $(this).data("data");

            var subj = data.otherNode;

            var leafPolicyVal = $('input[name=new_leaf_policy]:checked').val();
            var leafPolicy = undefined;
            switch (leafPolicyVal) {
              case "del":
                leafPolicy = gmClient.NEW_LEAF_POLICY_DEL;
                break;
              case "keep":
                leafPolicy = gmClient.NEW_LEAF_POLICY_KEEP;
                break;
              default:
                throw new Error("wrong leaf policy: " + leafPolicyVal);
                break;
            }

            // Disable "Move" button
            $("#move-button-ok").button("disable");

            gmClientLoggedIn.updateTag(subj.key, {
              parentTagID: data.node.key,
              newLeafPolicy: leafPolicy,
            }, function(status, resp) {
              // Enable "Move" button back
              $("#move-button-ok").button("enable");
              if (status == 200) {
                // move succeeded, so move the node visually, and close the
                // dialog
                subj.moveTo(node, data.hitMode);
                subj.makeVisible();
                $( self ).dialog( "close" );
              } else {
                // TODO: show error
                alert(JSON.stringify(resp));
              }

              $(data.node.span).removeClass("pending");
            });

          }
        },
        {
          id: "move-button-cancel",
          text: "Cancel",
          click: function() {
            $( this ).dialog( "close" );
          }
        },
      ],
      autoOpen: false,
      modal: true,
      minWidth: 400,
      maxHeight: 300,
      title: "Move tag",
    });
    moveDialog.on("dialogopen", function(event, ui) {
      var node = $(this).data("node");
      var data = $(this).data("data");
      var subj = data.otherNode;
      moveDialog.dialog(
        "option", "title",
        'Move "' + subj.title + '" under "' + data.node.title + '"'
      );
      $('input:radio[name=new_leaf_policy]')
        .filter('[value=del]')
        .prop('checked', true);
    });

    moveDialog.find('#move_dialog_details_link').click(function() {
      moveDialog.find('#move_dialog_details').toggle();
      return false;
    });
    // }}}

    // Setup tag deletion dialog {{{
    delDialog = contentElem.find('#del_dialog')
    delDialog.dialog({
      dialogClass: "no-close",
      buttons: [
        {
          id: "del-button-ok",
          text: "Delete!",
          click: function() {
            var self = this;

            var node = $(this).data("node");

            var leafPolicyVal = $('input[name=new_leaf_policy]:checked').val();
            var leafPolicy = undefined;
            switch (leafPolicyVal) {
              case "del":
                leafPolicy = gmClient.NEW_LEAF_POLICY_DEL;
                break;
              case "keep":
                leafPolicy = gmClient.NEW_LEAF_POLICY_KEEP;
                break;
              default:
                throw new Error("wrong leaf policy: " + leafPolicyVal);
                break;
            }

            // Disable "Delete" button
            $("#del-button-ok").button("disable");

            gmClientLoggedIn.deleteTag(
              node.key,
              leafPolicy,
              function(status, resp) {
                // Enable "Delete" button back
                $("#del-button-ok").button("enable");
                if (status == 200) {
                  // Deletion succeeded, so delete the node visually, and close
                  // the dialog
                  node.remove();
                  $( self ).dialog( "close" );
                } else {
                  // TODO: show error
                  alert(JSON.stringify(resp));
                }
              }
            );

          }
        },
        {
          id: "del-button-cancel",
          text: "Cancel",
          click: function() {
            $( this ).dialog( "close" );
          }
        },
      ],
      autoOpen: false,
      modal: true,
      minWidth: 400,
      maxHeight: 300,
      title: "Delete tag",
    });
    delDialog.on("dialogopen", function(event, ui) {
      var node = $(this).data("node");
      delDialog.dialog(
        "option", "title",
        'Delete "' + node.title + '"'
      );
      $('input:radio[name=new_leaf_policy]')
        .filter('[value=keep]')
        .prop('checked', true);
    });

    delDialog.find('#del_dialog_details_link').click(function() {
      delDialog.find('#del_dialog_details').toggle();
      return false;
    });
    // }}}

    _gmClient.createGMClientLoggedIn().then(function(instance) {
      if (instance) {
        initLoggedIn(instance, contentElem, srcDir);
      } else {
        gmPageWrapper.openPageLogin("openPageTagsTree", []);
        gmPageWrapper.closeCurrentWindow();
      }
    });

  }

  function initLoggedIn(instance, contentElem, srcDir) {
    gmClientLoggedIn = instance;
    var tagsTreeDiv = contentElem.find('#tags_tree_div')

    gmClientLoggedIn.getTagsTree(function(status, resp) {
      if (status == 200) {
        var treeData = convertTreeData(resp, true);

        tagsTreeDiv.fancytree({
          extensions: ["edit", "table", "dnd"],
          edit: {
            adjustWidthOfs: 4,   // null: don't adjust input size to content
            inputCss: { minWidth: "3em" },
            triggerStart: ["f2", "shift+click", "mac+enter"],
            beforeEdit: function(event, data) {
              if (data.node.key === rootTagKey) {
                return false;
              }
              return true;
            },
            edit: $.noop,        // Editor was opened (available as data.input)
            beforeClose: $.noop, // Return false to prevent cancel/save (data.input is available)
            save: saveTag,       // Save data.input.val() or return false to keep editor open
            close: $.noop,       // Editor was removed
          },
          table: {
          },
          dnd: {
            // Available options with their default:
            autoExpandMS: 1000,   // Expand nodes after n milliseconds of hovering
            draggable: null,      // Additional options passed to jQuery UI draggable
            droppable: null,      // Additional options passed to jQuery UI droppable
            focusOnClick: false,  // Focus, although draggable cancels mousedown event (#270)
            preventRecursiveMoves: true, // Prevent dropping nodes on own descendants
            preventVoidMoves: true,      // Prevent dropping nodes 'before self', etc.
            smartRevert: true,    // set draggable.revert = true if drop was rejected

            // Events that make tree nodes draggable
            dragStart: function(node, data) {
              return true;
            },
            dragStop: null,       // Callback(sourceNode, data)
            initHelper: null,     // Callback(sourceNode, data)
            updateHelper: null,   // Callback(sourceNode, data)

            // Events that make tree nodes accept draggables
            dragEnter: function(node, data) {
              // allow only moving nodes under other nodes; do not allow
              // reordering.
              // (to allow reordering, the returned array should also contain
              // "before", "after".)
              return ["over"];
            },
            dragExpand: null,     // Callback(targetNode, data), return false to prevent autoExpand
            dragOver: null,       // Callback(targetNode, data)
            dragDrop: function(node, data) {
              // This function MUST be defined to enable dropping of items on the tree.
              // data.hitMode is 'before', 'after', or 'over'.
              // We could for example move the source to the new target:
              var oldParent = data.otherNode.parent;
              var subj = data.otherNode;

              // Open the move dialog
              moveDialog
                .data("node", node)
                .data("data", data)
                .dialog("open");
            },
            dragLeave: null       // Callback(targetNode, data)
          },
          source: {
            children: [treeData],
          },
          renderColumns: function(event, data) {
            if (data.node.key !== rootTagKey) {
              var node = data.node;
              var $tdList = $(node.tr).find(">td");
              var $mainCol = $tdList.eq(0);

              // Add controls if not already
              if (!$mainCol.data("ctrlAdded")) {
                $mainCol.data("ctrlAdded", true);

                var $ctrlSpan = $("<span/>", {
                  class: "hidden",
                  id: "ctrl_" + data.node.key,
                });

                // Add "edit" link
                $("<a/>", {
                  href: "#",
                  text: "[edit]",
                  click: function() {
                    gmPageWrapper.openPageEditTag(data.node.key);
                  },
                }).appendTo($ctrlSpan);

                // Add "delete" link
                $("<a/>", {
                  href: "#",
                  text: "[x]",
                  class: "delete",
                  click: function() {
                    // Open the delete dialog
                    delDialog
                      .data("node", data.node)
                      .dialog("open");
                  },
                }).appendTo($ctrlSpan);

                $ctrlSpan.appendTo($mainCol.find(".fancytree-node"));

              }

              $tdList.mouseover(function() {
                $tdList.find("#ctrl_" + data.node.key).show();
              })

              $tdList.mouseout(function() {
                $tdList.find("#ctrl_" + data.node.key).hide();
              })
            }
          },
        });

        var tree = tagsTreeDiv.fancytree("getTree");
        var rootNode = tree.getNodeByKey(rootTagKey);
        rootNode.setExpanded(true);
      } else {
        // TODO: show error
        alert(JSON.stringify(resp));
      }
    })
  }

  function convertTreeData(tagsTree, isRoot) {
    var key = String(tagsTree.id);

    var ret = {
      title: tagsTree.names.join(", "),
      key: key,
    };

    keyToTag[key] = tagsTree;

    if (isRoot) {
      ret.title = "my tags";
      rootTagKey = key;
    }
    if ("subtags" in tagsTree) {
      ret.children = tagsTree.subtags.map(function(a) {
        return convertTreeData(a, false);
      });
      ret.folder = true;
    }
    return ret;
  }

  // see https://github.com/mar10/fancytree/wiki/ExtEdit for argument details
  function saveTag(event, data) {
    $(data.node.span).addClass("pending");
    var val = data.input.val();
    var prevVal = data.orgTitle;
    //console.log('saveTag event', event)
    //console.log('saveTag data', data)

    gmClientLoggedIn.updateTag(String(data.node.key), {
      names: val.split(",").map(function(a) {
        return a.trim();
      }),
    }, function(status, resp) {
      if (status == 200) {
        // update succeeded, do nothing here
      } else {
        // TODO: show error
        alert(JSON.stringify(resp));
        data.node.setTitle(prevVal);
      }

      $(data.node.span).removeClass("pending");
    });

    // Optimistically assume that save will succeed. Accept the user input
    return true;
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmTagsTree']={} : exports);
