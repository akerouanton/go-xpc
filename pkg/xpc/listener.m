#import "listener.h"

new_listener_res_t new_listener(const char *service, const char *requirement, uintptr_t opaque) {
	xpc_rich_error_t error;
	dispatch_queue_t queue = dispatch_queue_create(service, DISPATCH_QUEUE_CONCURRENT);

	xpc_listener_t listener = xpc_listener_create(
		service,
		queue,
		// We need to set the listener to inactive, otherwise it will start
		// processing messages before we have a chance to set code signing
		// requirements.
		XPC_LISTENER_CREATE_INACTIVE,
		^(xpc_session_t _Nonnull peer) {
			xpc_session_set_incoming_message_handler(peer, ^(xpc_object_t _Nonnull message) {
				on_msg_recv(opaque, peer, message);
			});
		},
		&error);

	if (listener == NULL) {
		dispatch_release(queue);
		return (new_listener_res_t){
			.err = error,
			.err_code = XPC_LISTENER_CREATE_FAILED,
		};
	}

	if (requirement != NULL) {
		if (xpc_listener_set_peer_code_signing_requirement(listener, requirement)) {
			xpc_listener_cancel(listener);
			xpc_release((xpc_object_t)listener);
			dispatch_release(queue);

			return (new_listener_res_t){
				.err_code = XPC_LISTENER_SET_PEER_CODE_SIGNING_REQUIREMENT_FAILED,
			};
		}
	}

	if (!xpc_listener_activate(listener, &error)) {
		xpc_listener_cancel(listener);
		xpc_release((xpc_object_t)listener);
		dispatch_release(queue);

		return (new_listener_res_t){
			.err = error,
			.err_code = XPC_LISTENER_ACTIVATE_FAILED,
		};
	}

	return (new_listener_res_t){
		.listener = listener,
		.queue = queue,
	};
}
