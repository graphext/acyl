let apiKeyID = "";

// showHideAPIKeysCreateNew toggles views based on render api keys response
function showHideAPIKeysCreateNew() {
    if (document.getElementById("apiKeysBody").hasChildNodes()) {
        document.getElementById("noAPIKeysCreateNew").setAttribute("class", "row hidden");
        document.getElementById("apiKeys").setAttribute("class", "row");
        document.getElementById("apiDocsHeading").setAttribute("class", "row");
    } else {
        document.getElementById("noAPIKeysCreateNew").setAttribute("class", "row");
        document.getElementById("apiKeys").setAttribute("class", "row hidden");
        document.getElementById("apiDocsHeading").setAttribute("class", "row hidden");
    }
}

// setAPIKeys replaces the existing api keys body
function setAPIKeys(apiKeysBody) {
    if (apiKeysBody === null) {
        return;
    }
    let oldAPIKeysBody = document.getElementById("apiKeysBody");
    oldAPIKeysBody.parentNode.replaceChild(apiKeysBody, oldAPIKeysBody);
    showHideAPIKeysCreateNew();
}

// renderCardHeader populates the api key headers
function renderCardHeader(apiKey) {
    let apiKeyHeadingRow = document.createElement('div');
    apiKeyHeadingRow.className = 'row';

    let caretIcon = document.createElement("span");
    caretIcon.className = "fas fa";
    caretIcon.innerHTML = "&#xf0d7;";

    let indicator = document.createElement("div");
    indicator.className = "col-sm-1";
    indicator.appendChild(caretIcon);
    apiKeyHeadingRow.appendChild(indicator);

    let description = document.createElement("div");
    description.className = "col-sm-11 text-left pl-4";
    description.innerHTML = apiKey.description;
    apiKeyHeadingRow.appendChild(description);

    let h5 = document.createElement('h5');
    h5.className = 'mb-0';
    h5.appendChild(apiKeyHeadingRow);

    let apiKeyBtn = document.createElement('button');
    apiKeyBtn.className = "collapsed btn btn-link text-dark btn-block text-decoration-none px-0";
    apiKeyBtn.type = "button";
    apiKeyBtn.setAttribute("data-toggle", "collapse");
    apiKeyBtn.setAttribute("data-target", `#cardBodyToggle-${apiKey.id}`);
    apiKeyBtn.setAttribute("aria-expanded", "false");
    apiKeyBtn.setAttribute("aria-controls", `cardBodyToggle-${apiKey.id}`);
    apiKeyBtn.appendChild(h5);

    let apiKeyCol = document.createElement('div');
    apiKeyCol.className = "col-sm-11 px-0";
    apiKeyCol.id = "apiKeyColumn";
    apiKeyCol.appendChild(apiKeyBtn);

    let deleteIcon = document.createElement('span');
    deleteIcon.className = "fas";
    deleteIcon.innerHTML = "&#xf2ed;";

    let deleteAPIKeyBtn = document.createElement('button');
    deleteAPIKeyBtn.className = "btn btn-link text-dark btn btn-block text-decoration-none pt-2";
    deleteAPIKeyBtn.id = `deleteAPIKeyButton-${apiKey.id}`;
    deleteAPIKeyBtn.type = "button";
    deleteAPIKeyBtn.onclick = (function(){
        apiKeyID = apiKey.id;
    });
    deleteAPIKeyBtn.setAttribute("data-toggle", "modal");
    deleteAPIKeyBtn.setAttribute("data-target", `#deleteConfirmationModal`);
    deleteAPIKeyBtn.setAttribute("data-backdrop", "static");
    deleteAPIKeyBtn.setAttribute("data-keyboard", "false");
    deleteAPIKeyBtn.setAttribute("title", "Delete API Key");
    deleteAPIKeyBtn.appendChild(deleteIcon);

    let deleteAPIKeyCol = document.createElement('div');
    deleteAPIKeyCol.className = "col-sm-1 px-0";
    deleteAPIKeyCol.id = "deleteAPIKeyColumn";
    deleteAPIKeyCol.appendChild(deleteAPIKeyBtn);

    let apiKeyRow = document.createElement('div');
    apiKeyRow.className = 'row';
    apiKeyRow.appendChild(apiKeyCol);
    apiKeyRow.appendChild(deleteAPIKeyCol);

    return apiKeyRow
}

// renderCardBodyTableBodyRow populates the table within the card body
function renderCardBodyTableBodyRow(heading, value){
    let th = document.createElement("th");
    th.scope = "row";
    th.innerHTML = heading;

    let td = document.createElement("td");
    td.id = heading.replace(/ /g,"_").toLowerCase();
    td.innerHTML = value;

    let tr = document.createElement("tr");
    tr.appendChild(th);
    tr.appendChild(td);

    return tr
}

// permissionLevel returns permission level mapping of integer:string
function permissionLevel() {
    return {
        0: "Unknown",
        1: "Read",
        2: "Write",
        3: "Admin",
    }
}

// encodePermissionLevel encodes the permission level string to integer
function permissionLevelEncode(permission) {
    if (permission === "Write") {
        return 2;
    }
    return 1;
}

function lastUsed(lu) {
    if (lu.hasOwnProperty('Valid') && lu.Valid === true) {
        return lu.Time;
    }
    return 'n/a';
}

// renderCollapse populates the collapsed details panel
function renderCollapse(apiKey) {
    let details = document.createElement("div");
    details.className = "col-sm pl-2";
    details.innerHTML = "Details";

    let tb = document.createElement("tbody");
    tb.appendChild(renderCardBodyTableBodyRow("Permission", permissionLevel()[apiKey.permission]));
    tb.appendChild(renderCardBodyTableBodyRow("Created", apiKey.created));
    tb.appendChild(renderCardBodyTableBodyRow("Last Used", lastUsed(apiKey.last_used)));

    let table = document.createElement("table");
    table.className = "table table-sm table-dark table-striped table-hover";
    table.id = `apiKeyDetails-${apiKey.id}`;
    table.appendChild(tb);

    let cb = document.createElement("div");
    cb.className = "card-body pb-0 py-1 px-3";
    cb.appendChild(details);
    cb.appendChild(table);

    return cb
}

// renderAPIKeys populates the new api keys body
function renderAPIKeys(apiKeys) {
    let apiKeysBody = document.createElement("div");
    apiKeysBody.className = "accordion";
    apiKeysBody.id = "apiKeysBody";
    if (apiKeys.length > 0) {
        for (let i = 0; i < apiKeys.length; i++) {
            let card = document.createElement('card');
            if (apiKeys[i].hasOwnProperty('id')) {
                let cardHeader = document.createElement('div');
                cardHeader.className = "card-header";
                cardHeader.id = `cardHeading-${apiKeys[i].id}`;
                cardHeader.appendChild(renderCardHeader(apiKeys[i]));
                card.appendChild(cardHeader);

                let collapse = document.createElement("div");
                collapse.id = `cardBodyToggle-${apiKeys[i].id}`;
                collapse.className = "collapse";
                collapse.setAttribute("aria-labelledby", `cardHeading-${apiKeys[i].id}`);
                collapse.appendChild(renderCollapse(apiKeys[i]));
                card.appendChild(collapse);
                apiKeysBody.appendChild(card);
            }
        }
    }
    setAPIKeys(apiKeysBody);
}

// newAPIKeyObj retrieves the values defined to create a new API Key
function newAPIKeyObj() {
    let apiKeyObj = {};
    let description = document.getElementById("newAPIKeyDescription").value;
    let permission = document.getElementById("newAPIKeyPermissionSelect").value;
    if (permission !== undefined) {
        if (description === "") {
            description = "n/a";
        }
        apiKeyObj = {
            "description": description,
            "permission": permissionLevelEncode(permission),
        };
    }
    return apiKeyObj;
}

// renderTokenWithCopyBtn formats the token with button to copy to clipboard
function renderTokenWithCopyBtn(token) {
    let tkn = document.createElement('code');
    tkn.innerHTML = token;

    let tknCol = document.createElement('div');
    tknCol.className = "col-10 pl-5";
    tknCol.appendChild(tkn);

    const copyImg = `<img height="18" width="18" src="https://cdnjs.cloudflare.com/ajax/libs/octicons/8.5.0/svg/clippy.svg" alt="Copy to clipboard">`;
    let copyBtn = document.createElement('span');
    copyBtn.id = "token-copy-btn";
    copyBtn.setAttribute("style", "cursor: pointer");
    copyBtn.setAttribute("data-toggle", "tooltip");
    copyBtn.setAttribute("data-trigger", "click");
    copyBtn.title= "Copied!";
    copyBtn.innerHTML = copyImg;

    let copyCol = document.createElement('div');
    copyCol.className = "col-2 text-center";
    copyCol.appendChild(copyBtn);

    let tknRow = document.createElement('div');
    tknRow.className = "row pb-2";
    tknRow.setAttribute("style", "align-content: center");
    tknRow.appendChild(tknCol);
    tknRow.appendChild(copyCol);

    document.getElementById("newTokenID").appendChild(tknRow);
    $('#token-copy-btn').tooltip();
    document.getElementById("token-copy-btn").onclick = function(e) {
        e.preventDefault();
        navigator.clipboard.writeText(token);
    };
    $('#token-copy-btn').on('shown.bs.tooltip', function() {
        setTimeout(function() { $("#token-copy-btn").tooltip('hide'); }, 500);
    });
}

// setToken sets the token to be displayed upon creation within the create modal and configures the view
function setToken(token) {
    clearTokenError();
    document.getElementById("newTokenForm").setAttribute("class", "row hidden");
    document.getElementById("newAPIKeyConfirmationModalCreate").setAttribute("class", "btn btn-primary hidden");
    renderTokenWithCopyBtn(token);
    document.getElementById("newTokenAlert").setAttribute("class", "row");
}

// clearToken clears the token from within the presented modal and configures the view
function clearToken() {
    if (document.getElementById("newTokenID").innerText !== "") {
        document.getElementById("newTokenAlert").setAttribute("class", "row hidden");
        document.getElementById("newTokenID").innerText = "";
        document.getElementById("newAPIKeyConfirmationModalCreate").setAttribute("class", "btn btn-primary");
        document.getElementById("newTokenForm").setAttribute("class", "row");
    }
    clearTokenError();
}

// setTokenError sets the token to be displayed upon creation within the create modal and configures the view
function setTokenError(error) {
    document.getElementById("newTokenError").innerText = error;
    document.getElementById("newTokenErrorAlert").setAttribute("class", "row");
}

// clearTokenError clears the error from within the presented modal and configures the view
function clearTokenError() {
    if (document.getElementById("newTokenError").innerText !== "") {
        document.getElementById("newTokenError").innerText = "";
        document.getElementById("newTokenErrorAlert").setAttribute("class", "row hidden");
    }
}

// setDeleteError sets the error to be displayed within the delete modal
function setDeleteError(error) {
    document.getElementById("deleteConfirmationModalDelete").removeAttribute("data-dismiss");
    document.getElementById("deleteTokenError").innerText = error;
    document.getElementById("deleteTokenErrorAlert").setAttribute("class", "row");
    document.getElementById("deleteConfirmationModalDelete").setAttribute("class", "btn btn-primary hidden");
}

// clearDeleteError clears the error from within the presented modal
function clearDeleteError() {
    if (document.getElementById("deleteTokenError").innerText !== "") {
        document.getElementById("deleteTokenError").innerText = "";
        document.getElementById("deleteTokenErrorAlert").setAttribute("class", "row hidden");
        document.getElementById("deleteConfirmationModalDelete").setAttribute("class", "btn btn-primary");
        document.getElementById("deleteConfirmationModalDelete").setAttribute("data-dismiss", "modal");
    }
}

document.addEventListener("DOMContentLoaded", function(){
    getAPIKeys();
    $('#newAPIKeyConfirmationModalCreate').on('click', function () {
        createAPIKey(newAPIKeyObj());
    });
    $('#newAPIKeyConfirmationModalClose').on('click', function () {
        clearToken();
    });
    $('#newAPIKeyConfirmationModalCloseMain').on('click', function () {
        clearToken();
    });
    $('#deleteConfirmationModalClose').on('click', function () {
        apiKeyID = "";
        clearDeleteError()
    });
    $('#deleteConfirmationModalCloseMain').on('click', function () {
        apiKeyID = "";
        clearDeleteError()
    });
    $('#deleteConfirmationModalDelete').on('click', function () {
        deleteAPIKey(apiKeyID);
    });
});

function getAPIKeys() {
    let req = new XMLHttpRequest();

    req.open('GET', `${apiBaseURL}/v2/user/tokens`, true);
    req.onload = function () {
        if (req.status !== 200) {
            console.log(`user tokens request failed: ${req.status}: ${req.responseText}`);
            return;
        }
        const data = JSON.parse(req.response);
        renderAPIKeys(data);
    };
    req.onerror = function () {
        console.error(`error getting user tokens: ${req.statusText}`);
    };
    req.send(null);
}

function createAPIKey(data) {
    let req = new XMLHttpRequest();

    req.open('POST', `${apiBaseURL}/v2/user/token`, true);
    req.onload = function () {
        if (req.status !== 201) {
            let err = `create user token request failed: ${req.status}`;
            if (req.responseText !== "") {
                err = err.concat(`: ${req.responseText}`);
            }
            if (req.status === 429) {
                err = "api key limit reached"
            }
            console.log(err);
            setTokenError(err);
            return;
        }
        const data = JSON.parse(req.response);
        if (data !== null) {
            if (data.hasOwnProperty('token')) {
                setToken(data.token);
                getAPIKeys();
            }
        }
    };
    req.onerror = function () {
        let err = `error creating user token: ${req.status}`;
        if (req.responseText !== "") {
            err = err.concat(`: ${req.responseText}`);
        }
        console.log(err);
        setTokenError(err);
    };
    req.send(JSON.stringify(data));
}

function deleteAPIKey(apiKeyID) {
    let req = new XMLHttpRequest();

    req.open('DELETE', `${apiBaseURL}/v2/user/token/${apiKeyID}`, true);
    req.onload = function () {
        if (req.status !== 204) {
            let err = `delete user token request failed: ${req.status}`;
            if (req.responseText !== "") {
                err = err.concat(`: ${req.responseText}`);
            }
            console.log(err);
            setDeleteError(err);
            return;
        }
        getAPIKeys();
    };
    req.onerror = function () {
        let err = `error deleting user token: ${req.status}`;
        if (req.responseText !== "") {
            err = err.concat(`: ${req.responseText}`);
        }
        console.log(err);
        setDeleteError(err);
    };
    req.send(null);
}
