import React, { useEffect, useState } from "react";
import { useMutation } from "@connectrpc/connect-query";
import { adminRedeemOneTimeToken } from "@/gen/ssoready/v1/ssoready-SSOReadyService_connectquery";
import { useNavigate, useSearchParams } from "react-router-dom";
import { setSessionToken } from "@/auth";

export function SetupPage() {
  const redeemOneTimeTokenMutation = useMutation(adminRedeemOneTimeToken);
  const [searchParams] = useSearchParams();
  const oneTimeToken = searchParams.get("one-time-token") ?? undefined;
  const navigate = useNavigate();
  const [error, setError] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const { adminSessionToken } =
          await redeemOneTimeTokenMutation.mutateAsync({
            oneTimeToken,
          });

        setSessionToken(adminSessionToken);
        navigate("/");
      } catch {
        // Expected error when token is invalid or already used
        setError(true);
      }
    })();
  }, [oneTimeToken, redeemOneTimeTokenMutation.mutateAsync, navigate]);

  if (error) {
    return (
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <p>
          This setup link has already been used or is invalid. Setup links
          expire once you visit them.
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <p>Loading...</p>
    </div>
  );
}
