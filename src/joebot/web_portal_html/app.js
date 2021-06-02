"use strict";

let clients = [];

new Vue({
	el: '#clients_list',

	data: {
		items: clients,
		fields: [
			{ key: 'id', label: 'ID', sortable: true, tdAttr: {style:"width:9%;word-break:break-all;word-wrap:break-word;"} },
			{ key: 'ip', label: 'IP', sortable: true, tdAttr: {style:"width:8%;word-break:break-all;word-wrap:break-word;"} },
			{ key: 'host_name', label: 'Hostname', sortable: true, tdAttr: {style:"width:10%;word-break:break-all;word-wrap:break-word;"} },
			{ key: 'username', label: 'User', sortable: true, tdAttr: {style:"width:5%;word-break:break-all;word-wrap:break-word;"} },
			{ key: 'port_tunnels', label: 'Port Tunnels', sortable: false, tdAttr: {style:"width:15%;word-break:break-all;word-wrap:break-word;"} },
			{ key: 'tags', label: 'Tags', sortable: false, tdAttr: {style:"width:30%;word-break:break-all;word-wrap:break-word;"} },
			{ key: 'actions', label: 'Actions', tdAttr: {style:"width:23%;word-break:break-all;word-wrap:break-word;"} }
		],
		currentPage: 1,
		perPage: 99999,
		totalRows: clients.length,
		sortBy: null,
		sortDesc: false,
		filter: null,
		modalInfo: { title: '', content: '' },
		
		filteredItems: [],
		selected: [],
		selectAll: false,

		targetClientId: null,
		targetClientPortToBeCreated: null,

		targetJoebotServerAddr: null,
		targetIPList: null,
		targetSSHUser: null,
		targetSSHPassword: null,
		targetSSHKeyContent: null,
	},
	
	created: function() {
		const updateTable = () => {
			this.$http.get('/api/clients').then(result => {
				let clients = result.body.clients;
				let newClientIDs = clients.map(client => client.id);
				
				//Remove existing entries
				this.items = this.items.filter(item => newClientIDs.indexOf(item.id) >= 0);
				
				//Update existing entries
				this.items.forEach((item, index) => {
					let obj = clients.find(client => client.id == item.id);
					Object.keys(obj).forEach((key, index) => {
						item[key] = obj[key];
					});
				});
				
				// Append new clients
				clients.forEach(client => {
					let itemIds = this.items.map(item => item.id);
					if( itemIds.indexOf(client.id) < 0 ){
						this.items.push(client);
					}
				});
				
				this.totalRows = this.items.length;
			});
		};
		
		let url = window.location.href;
		let captured = /filter=([^&]+)/.exec(url);
		if (captured){
			this.filter = captured[1] ? decodeURIComponent(captured[1]) : null;
		}

		updateTable();
		setInterval(() => updateTable(), 1000);
	},
	
	methods: {
		select () {
			let visibleItemIDs = this.filteredItems.map(item => item.id);
			this.selected = [];
			if (!this.selectAll) {
				for (let i in this.items) {
					if (this.filteredItems.length === 0 || visibleItemIDs.includes(this.items[i].id))
						this.selected.push(this.items[i].id);
				}
			}
		},
		info (item, index, button) {
			this.modalInfo.title = item.id;
			this.modalInfo.content = JSON.stringify(item, null, 3);
			this.$root.$emit('bv::show::modal', 'modalInfo', button);
		},
		open_terminal (item) {
			if( item.gotty_web_terminal_info ){
				window.open(`http://${window.location.hostname}:${item.gotty_web_terminal_info.port_tunnel.server_port}`);
			}
		},
		open_filebrowser (item) {
			if( item.filebrowser_info ){
				var postURL = "files" + encodeURI(item.filebrowser_info.default_directory);
				window.open(`http://${window.location.hostname}:${item.filebrowser_info.port_tunnel.server_port}/${postURL}`);
			}
		},
		open_vnc (item) {
			if( item.novnc_websocket_info ){
				window.open(`http://novnc.com/noVNC/vnc.html?host=${window.location.hostname}&port=${item.novnc_websocket_info.port_tunnel.server_port}&encrypt=0&autoconnect=1`);
			}
		},
		resetModal () {
			this.modalInfo.title = '';
			this.modalInfo.content = '';
		},
		onFiltered (filteredItems) {
			this.filteredItems = filteredItems;
			this.totalRows = filteredItems.length;
			this.currentPage = 1;
		},
		
		bulk_install () {
			this.targetJoebotServerAddr = location.hostname + ":13579",
			this.targetIPList = "";
			this.targetSSHUser = "";
			this.targetSSHPassword = "";
			this.targetSSHKeyContent = "";
			this.$refs.modalInitBulkInstall.show();
		},
		handleInitBulkInstallOk (evt) {
			console.log("handleInitBulkInstallOk");

			let bulkInstallInfo = {
				'JoebotServerIP': null,
				'JoebotServerPort': 13579,
				'Addresses': [],
				'Username': this.targetSSHUser,
				'Password': this.targetSSHPassword,
				'Key': this.targetSSHKeyContent
			};

			if (this.targetJoebotServerAddr === ''){
				alert('Missing joebot server address');
				return
			}
			if (this.targetSSHUser === ''){
				alert('SSH username is missing');
				return;
			}
			if (this.targetSSHPassword === '' && this.targetSSHKeyContent === ''){
				alert('Either SSH password or key must be defined');
				return;
			}

			let joebotAddr = this.targetJoebotServerAddr.split(':');
			bulkInstallInfo.JoebotServerIP = joebotAddr[0];
			bulkInstallInfo.JoebotServerPort = (joebotAddr.length===2)?parseInt(joebotAddr[1]):bulkInstallInfo.JoebotServerPort;

			let addresses = this.targetIPList.split('\n');
			for (let address of addresses) {
				address = address.trim();
				if (address === '')
					continue;

				let tmp = address.trim().split(':');
				let ip = tmp[0]
				let port = (tmp.length === 2)?parseInt(tmp[1]):22;

				bulkInstallInfo['Addresses'].push({
					'IP': ip,
					'Port': port
				});
			}

			if (bulkInstallInfo['Addresses'].length === 0){
				alert('Address list is empty');
				return;
			}

			this.handleSubmitInitBulkInstall(bulkInstallInfo);
		},
		handleSubmitInitBulkInstall (bulkInstallInfo) {
			console.log("handleSubmitInitBulkInstall");

			console.log(JSON.stringify(bulkInstallInfo, null, 3));
			axios.post('/api/bulk-install', bulkInstallInfo)
			.then(function (response) {
				console.log(response);
			})
			.catch(function (error) {
				console.log(error);
			});

			this.$refs.modalInitBulkInstall.hide();
		},
		focusTargetIPList () {
			this.$refs.modalTargetIPList.focus();
		},

		create_tunnel (item) {
			this.targetClientId = item.id;
			this.$refs.modalCreateTunnel.show();
		},
		focusInputPort () {
			this.$refs.modalTargetPortInput.focus();
		},
		handleCreateTunnelOk (evt) {
			if (!this.targetClientPortToBeCreated || parseInt(this.targetClientPortToBeCreated) <= 0) {
				evt.preventDefault();
				this.targetClientPortToBeCreated = null;
				alert('Please enter a valid client port');
			} else {
				this.handleSubmitTunnelCreation();
			}
		},
		handleSubmitTunnelCreation () {
			let data = new FormData();
			data.set('target_client_port', parseInt(this.targetClientPortToBeCreated))
			this.$http.post(`/api/client/${this.targetClientId}`, data).then(response => {
				console.log(`Created Tunnel: \n${JSON.stringify(response.body, null, 3)}`);
			});
			
			this.$refs.modalCreateTunnel.hide();
			this.targetClientId = null;
			this.targetClientPortToBeCreated = null;
		}
	}
});