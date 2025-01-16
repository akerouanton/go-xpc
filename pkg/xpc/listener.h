#import <xpc/xpc.h>

#define XPC_LISTENER_CREATE_FAILED -1
#define XPC_LISTENER_SET_PEER_CODE_SIGNING_REQUIREMENT_FAILED -2
#define XPC_LISTENER_ACTIVATE_FAILED -3

typedef struct new_listener_res_t {
    xpc_listener_t listener;
	dispatch_queue_t queue;
    xpc_rich_error_t err;
	int err_code;
} new_listener_res_t;

new_listener_res_t new_listener(const char *service, const char *requirement, uintptr_t opaque);

extern void on_msg_recv(uintptr_t opaque, xpc_session_t peer, xpc_object_t msg);
