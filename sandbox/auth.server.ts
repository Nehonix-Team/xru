        import { MultiServerConfig } from "xypriss";
        import { xmsc } from "../configs/xms.config";

        /**
        * Authentication Server Configuration
        *
        * Defines the operational parameters for the isolated authentication instance.
        * This includes identity, port binding, and specific orchestration rules
        * such as route prefixing and cluster activation.
        */
        export const authServer: MultiServerConfig = {
        /** Unique identifier for the authentication service */
        id: xmsc.auth.id,
        /** Network port assignment */
        port: xmsc.auth.port,
        /** URL path scope for this server instance */
        routePrefix: "/auth",
        /** Horizontal scaling configuration */
        cluster: {
            /**
            * Indicates whether the server should run in a multi-process cluster mode.
            * Recommendation: Enable in production environments for high availability.
            */
            enabled: false,
        },
        };
