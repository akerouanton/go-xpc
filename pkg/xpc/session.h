#import <xpc/xpc.h>

typedef struct new_session_res_t {
    xpc_session_t session;
	dispatch_queue_t queue;
    xpc_rich_error_t err;
} new_session_res_t;

new_session_res_t new_session(const char *service);

typedef struct send_reply_t {
	xpc_object_t reply;
	xpc_rich_error_t err;
} send_reply_t;

send_reply_t send_message_with_reply(xpc_session_t session, xpc_object_t payload);
