#import "session.h"

new_session_res_t new_session(const char *service) {
	xpc_rich_error_t error;
	dispatch_queue_t queue = dispatch_queue_create(service, DISPATCH_QUEUE_CONCURRENT);

	xpc_session_t session = xpc_session_create_mach_service(service, queue, XPC_SESSION_CREATE_MACH_PRIVILEGED, &error);
	if (session == NULL) {
		return (new_session_res_t){
			.err = error,
		};
	}

	return (new_session_res_t){
		.session = session,
		.queue = queue,
	};
}

send_reply_t send_message_with_reply(xpc_session_t session, xpc_object_t payload) {
	xpc_rich_error_t error;

	xpc_object_t reply = xpc_session_send_message_with_reply_sync(session, payload, &error);
	if (reply == NULL) {
		return (send_reply_t){
			.err = error,
		};
	}

	return (send_reply_t){
		.reply = reply,
	};
}
