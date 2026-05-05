/**
 * XyPriss Default Server Configuration
 *
 * Configured for a standard, single-instance server operation.
 */

import { __sys__ } from "xypriss";
import { manifest } from "./manifest";
 import { authServer } from "../servers/auth.server";
 import { mainServer } from "../servers/main.server";

export const serverConfigs: ServerOptions = {
   multiServer: {
       enabled: true,
       servers: [
           authServer,
           mainServer,
       ],
   },

};
