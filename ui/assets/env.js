function updateNSCopyBtn(k8s_ns) {
    if (k8s_ns === "") {
        document.getElementById("k8s-ns").innerHTML = "n/a";
        return;
    }
    if (document.getElementById("ns-copy-btn") === null) {
        const copybtn = `<span id="ns-copy-btn" style="cursor: pointer" data-toggle="tooltip" title="Copied!" data-trigger="click"><img class="nscopybtn" height="18" width="18" src="https://cdnjs.cloudflare.com/ajax/libs/octicons/8.5.0/svg/clippy.svg" alt="Copy to clipboard"></span>`;
        document.getElementById("k8s-ns").innerHTML = `${k8s_ns} ${copybtn}`;
        // have to use JQuery to manage the copy button tooltip
        $("#ns-copy-btn").tooltip();
        // copy namespace to clipboard button
        document.getElementById("ns-copy-btn").onclick = function(event) {
            event.preventDefault();
            navigator.clipboard.writeText(k8s_ns);
        };
        $("#ns-copy-btn").on('shown.bs.tooltip', function() {
            setTimeout(function() { $("#ns-copy-btn").tooltip('hide'); }, 500);
        });
    }
}

function renderDetailTable(env) {
    let envstatlabel = "";
    let envstatclasses = "";
    switch (env.status) {
        case "success":
            envstatlabel = "Success";
            envstatclasses = "badge badge-success";
            break;
        case "failed":
            envstatlabel = "Failed";
            envstatclasses = "badge badge-danger";
            break;
        case "pending":
            envstatlabel = "Pending";
            envstatclasses = "badge badge-warning";
            break;
        case "destroyed":
            envstatlabel = "Destroyed";
            envstatclasses = "badge badge-secondary";
            break;
        default:
            envstatlabel = "Unknown";
            envstatclasses = "badge badge-secondary";
            break;
    }
    document.getElementById("status-badge").innerHTML = envstatlabel;
    document.getElementById("status-badge").className = envstatclasses;
    document.getElementById("env-repo").innerHTML = `<a href="https://github.com/${env.repo}">https://github.com/${env.repo}</a>`;
    document.getElementById("env-pr-link").innerHTML = `<a href="https://github.com/${env.repo}/pull/${env.pull_request}">https://github.com/${env.repo}/pull/${env.pull_request}</a>`;
    document.getElementById("env-user-link").innerHTML = `<a href="https://github.com/${env.github_user}">${env.github_user}</a>`;
    document.getElementById("trepo-branch").innerHTML = env.pr_head_branch;
    updateNSCopyBtn(env.k8s_namespace);
}

// seteventlist replaces the table body with tbody
function seteventlist(tbody) {
    let oldtbody = document.getElementById("eventlist-tbody");
    if (tbody === null) {
        return;
    }
    oldtbody.parentNode.replaceChild(tbody, oldtbody);
    tbody.id = "eventlist-tbody";
}

function eventrow(event) {
    let tr = document.createElement("tr");
    let tdstarted = document.createElement("td");
    tdstarted.className = "text-left";
    tdstarted.innerHTML = event.started;
    tr.appendChild(tdstarted);
    let tdduration = document.createElement("td");
    tdduration.className = "text-left";
    if (event.duration === null) {
        tdduration.innerHTML = "";
    } else {
        tdduration.innerHTML = event.duration;
    }
    tr.appendChild(tdduration);
    let tdtype = document.createElement("td");
    tdtype.className = "text-left";
    switch (event.type) {
        case "create":
            tdtype.innerHTML = `<span class="fas">&#xf07c;</span> Opened`;
            break;
        case "update":
            tdtype.innerHTML = `<span class="fas">&#xf2f1;</span> Sync`;
            break;
        case "destroy":
            tdtype.innerHTML = `<span class="fas">&#xf2ed;</span> Closed`;
            break;
        default:
            tdtype.innerHTML = `<span class="fas">&#xf059;</span> Unknown`;
    }
    tr.appendChild(tdtype);
    let tdstatus = document.createElement("td");
    tdstatus.className = "text-center";
    tr.appendChild(tdstatus);
    // color whole rows on non-success envs
    switch (event.status) {
        case "done":
            tdstatus.innerHTML = `<span class="badge badge-success">Done</span>`;
            break;
        case "failed":
            tr.className = "table-danger";
            tdstatus.innerHTML = `<span class="badge badge-danger">Failed</span>`;
            break;
        case "pending":
            tr.className = "table-warning";
            tdstatus.innerHTML = `<span class="badge badge-warning">Pending</span>`;
            break;
        default:
            tr.className = "table-active";
            tdstatus.innerHTML = `<span class="badge badge-secondary">Unknown</span>`;
            break;
    }
    let tddetails = document.createElement("td");
    tddetails.className = "text-center";
    tddetails.innerHTML = `<a href="${apiBaseURL}/ui/event/status?id=${event.event_id}"><button type="button" class="fas btn btn-sm btn-outline-secondary">&#xf05a;</button></a>`;
    tr.appendChild(tddetails);
    return tr;
}

function renderEventList(events) {
    events.sort(function(x, y) {
        const date1 = new Date(x.started);
        const date2 = new Date(y.started);
        return date2 - date1;
    });
    let tbody = document.createElement("tbody");
    for (let i = 0; i < events.length; i++) {
        let row = eventrow(events[i]);
        tbody.appendChild(row);
    }
    seteventlist(tbody);
}

function renderEnvDetail(env) {
    renderDetailTable(env);
    renderEventList(env.events);
}

function update() {
    let req = new XMLHttpRequest();

    req.open('GET', `${apiBaseURL}/v2/userenvs/${envName}`, true);
    req.onload = function (e) {
        if (req.status !== 200) {
            console.log(`env detail request failed: ${req.status}: ${req.responseText}`);
            return;
        }

        const data = JSON.parse(req.response);
        renderEnvDetail(data);
    };
    req.onerror = function (e) {
        console.error(`error getting env detail endpoint: ${req.statusText}`);
    };
    req.send(null);
}

document.addEventListener("DOMContentLoaded", function(){
    document.getElementById("refreshbtn").addEventListener('click', function(e){
        e.preventDefault();
        update();
    });
    if (document.getElementById("synchronizeModal") !== null) {
        $("#synchronizeModal").on('shown.bs.modal', function () {
            $('#synchronizeModalConfirm').on('click', function (e) {
                e.preventDefault();
                rebuild();
                update();
            });
        });
    }
    if (document.getElementById("rebuildModal") !== null) {
        $("#rebuildModal").on('shown.bs.modal', function () {
            $('#rebuildModalConfirm').on('click', function (e) {
                e.preventDefault();
                rebuild(true);
                update();
            });
        });
    }
    update();
});

function rebuild(fullRebuild = false) {
    let req = new XMLHttpRequest();
    let message = fullRebuild ? 'rebuild' : 'synchronize';

    req.open('POST', `${apiBaseURL}/v2/userenvs/${envName}/actions/rebuild?full=${fullRebuild}`, false);
    req.onload = function () {
        if (req.status !== 201) {
            console.log(`env ${message} request failed: ${req.status}: ${req.responseText}`);
        }
        const data = JSON.parse(req.response);
        renderEnvDetail(data);
    };
    req.onerror = function () {
        console.error(`error rebuilding environment: ${req.statusText}`);
    };
    req.send(null);
}
